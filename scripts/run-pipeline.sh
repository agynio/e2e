#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
parse_script="$repo_root/scripts/parse_suite.py"
suites_dir="$repo_root/suites"
artifacts_dir="$repo_root/.artifacts/junit"
diagnostics_root="$repo_root/.diagnostics/suites"

tags="${TAGS:-}"
namespace="${E2E_NAMESPACE:-${DEVSPACE_NAMESPACE:-platform}}"
suites_filter="${E2E_SUITES:-}"

if [ ! -d "$suites_dir" ]; then
  echo "ERROR: suites directory not found at $suites_dir" >&2
  exit 1
fi

rm -rf "$artifacts_dir" "$diagnostics_root"
mkdir -p "$artifacts_dir" "$diagnostics_root"

mapfile -t suite_files < <(find "$suites_dir" -mindepth 2 -maxdepth 2 -name suite.yaml | sort)

if [ -n "$suites_filter" ]; then
  read -r -a requested_suites <<< "$suites_filter"
  filtered_suite_files=()
  missing_suites=()

  for requested_suite in "${requested_suites[@]}"; do
    found_suite=0
    for suite_file in "${suite_files[@]}"; do
      suite_name=$(basename "$(dirname "$suite_file")")
      if [ "$suite_name" = "$requested_suite" ]; then
        filtered_suite_files+=("$suite_file")
        found_suite=1
        break
      fi
    done
    if [ "$found_suite" -ne 1 ]; then
      missing_suites+=("$requested_suite")
    fi
  done

  if [ "${#missing_suites[@]}" -gt 0 ]; then
    printf 'ERROR: unknown E2E_SUITES entries: %s\n' "${missing_suites[*]}" >&2
    exit 1
  fi

  suite_files=("${filtered_suite_files[@]}")
fi

if [ "${#suite_files[@]}" -eq 0 ]; then
  echo "ERROR: No suites found under $suites_dir" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "ERROR: python3 not found in PATH" >&2
  exit 1
fi

if ! python3 -c 'import yaml' >/dev/null 2>&1; then
  echo "ERROR: PyYAML is required for suite parsing. Install with: python3 -m pip install pyyaml" >&2
  exit 1
fi

matched_suite_count=0
selected_suite_files=()

for suite_file in "${suite_files[@]}"; do
  suite_dir=$(dirname "$suite_file")
  suite_name=$(basename "$suite_dir")
  tmp_dir=$(mktemp -d)

  if ! python3 "$parse_script" "$suite_file" "$tmp_dir"; then
    rm -rf "$tmp_dir"
    echo "ERROR: failed to parse suite $suite_name" >&2
    exit 1
  fi

  if ! select_output=$(
    cd "$suite_dir"
    export TAGS="$tags"
    bash -s < "$tmp_dir/select"
  ); then
    rm -rf "$tmp_dir"
    echo "ERROR: select failed for suite $suite_name" >&2
    exit 1
  fi

  rm -rf "$tmp_dir"

  if [ -n "$(echo "$select_output" | tr -d '[:space:]')" ]; then
    selected_suite_files+=("$suite_file")
    matched_suite_count=$((matched_suite_count + 1))
  fi
done

if [ "$matched_suite_count" -eq 0 ] && [ -n "$(echo "$tags" | tr -d '[:space:]')" ]; then
  echo "ERROR: TAGS matched zero suites: $tags" >&2
  exit 1
fi

suite_files=("${selected_suite_files[@]}")

if [ "${#suite_files[@]}" -eq 0 ]; then
  echo "No suites selected."
  exit 0
fi

if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl not found in PATH" >&2
  exit 1
fi

if ! kubectl get namespace "$namespace" >/dev/null 2>&1; then
  kubectl create namespace "$namespace"
fi

overall_exit=0

for suite_file in "${suite_files[@]}"; do
  if ! (
    set -euo pipefail

    suite_dir=$(dirname "$suite_file")
    suite_name=$(basename "$suite_dir")
    suite_slug=$(echo "$suite_name" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-' | sed 's/^-//;s/-$//')
    tmp_dir=$(mktemp -d)
    pod_name=""
    applied_manifest_paths=()

    manifest_namespace() {
      local manifest_file=$1
      python3 - "$manifest_file" <<'PY'
import sys

import yaml

path = sys.argv[1]
try:
    with open(path, "r", encoding="utf-8") as handle:
        docs = list(yaml.safe_load_all(handle))
except Exception as exc:
    print(f"{exc}", file=sys.stderr)
    sys.exit(1)

for doc in docs:
    if isinstance(doc, dict):
        meta = doc.get("metadata") or {}
        namespace = meta.get("namespace")
        if isinstance(namespace, str) and namespace.strip():
            print(namespace.strip())
            break
PY
    }

    # shellcheck disable=SC2329
    cleanup() {
      local exit_code=$?
      set +e
      if [ -n "${pod_name:-}" ]; then
        kubectl delete pod "$pod_name" -n "$namespace" --wait=true --ignore-not-found
      fi
      if [ "${#applied_manifest_paths[@]}" -gt 0 ]; then
        for manifest_path in "${applied_manifest_paths[@]}"; do
          if manifest_ns=$(manifest_namespace "$manifest_path"); then
            if [ -n "$manifest_ns" ]; then
              kubectl delete -f "$manifest_path" --ignore-not-found
            else
              kubectl delete -n "$namespace" -f "$manifest_path" --ignore-not-found
            fi
          else
            echo "ERROR: failed to parse manifest $manifest_path during cleanup" >&2
          fi
        done
      fi
      if [ -n "${tmp_dir:-}" ]; then
        rm -rf "$tmp_dir"
      fi
      exit "$exit_code"
    }
    trap cleanup EXIT

    python3 "$parse_script" "$suite_file" "$tmp_dir"
    image=$(cat "$tmp_dir/image")
    workdir=$(cat "$tmp_dir/workdir")
    select_file="$tmp_dir/select"
    run_file="$tmp_dir/run"
    manifests_file="$tmp_dir/manifests"
    service_account_file="$tmp_dir/service_account"
    required_env_file="$tmp_dir/required_env"

    service_account=""
    if [ -s "$service_account_file" ]; then
      service_account=$(cat "$service_account_file" | xargs)
    fi

    manifests=()
    if [ -s "$manifests_file" ]; then
      while IFS= read -r manifest || [ -n "$manifest" ]; do
        trimmed=$(echo "$manifest" | xargs)
        if [ -n "$trimmed" ]; then
          manifests+=("$trimmed")
        fi
      done < "$manifests_file"
    fi

    if [ -z "$image" ]; then
      echo "ERROR: suite $suite_name missing image" >&2
      exit 1
    fi

    if [ -z "$workdir" ]; then
      workdir="/opt/app/data"
    fi

    if [ ! -s "$select_file" ]; then
      echo "ERROR: suite $suite_name missing select command" >&2
      exit 1
    fi

    if [ ! -s "$run_file" ]; then
      echo "ERROR: suite $suite_name missing run command" >&2
      exit 1
    fi

    required_envs=()
    if [ -s "$required_env_file" ]; then
      while IFS= read -r env_name || [ -n "$env_name" ]; do
        trimmed=$(echo "$env_name" | xargs)
        if [ -n "$trimmed" ]; then
          required_envs+=("$trimmed")
        fi
      done < "$required_env_file"
    fi

    if [ "${#required_envs[@]}" -gt 0 ]; then
      missing_envs=()
      for env_name in "${required_envs[@]}"; do
        env_value=${!env_name:-}
        trimmed_value=$(echo "$env_value" | xargs)
        if [ -z "$trimmed_value" ]; then
          missing_envs+=("$env_name")
        fi
      done
      if [ "${#missing_envs[@]}" -gt 0 ]; then
        printf 'ERROR: suite %s missing required environment variables: %s\n' \
          "$suite_name" "${missing_envs[*]}" >&2
        exit 1
      fi
    fi

    manifest_files=()
    if [ "${#manifests[@]}" -gt 0 ]; then
      for manifest in "${manifests[@]}"; do
        manifest_path="$suite_dir/$manifest"
        if [ ! -e "$manifest_path" ]; then
          echo "ERROR: suite $suite_name manifest not found at $manifest_path" >&2
          exit 1
        fi
        if [ -d "$manifest_path" ]; then
          mapfile -t manifest_dir_files < <(find "$manifest_path" -maxdepth 1 -type f \
            \( -name '*.yml' -o -name '*.yaml' -o -name '*.json' \) | sort)
          if [ "${#manifest_dir_files[@]}" -eq 0 ]; then
            echo "ERROR: suite $suite_name manifest directory empty at $manifest_path" >&2
            exit 1
          fi
          for manifest_file in "${manifest_dir_files[@]}"; do
            manifest_files+=("$manifest_file")
          done
        else
          manifest_files+=("$manifest_path")
        fi
      done
      for manifest_file in "${manifest_files[@]}"; do
        if ! manifest_ns=$(manifest_namespace "$manifest_file"); then
          echo "ERROR: suite $suite_name failed to parse manifest $manifest_file" >&2
          exit 1
        fi
        apply_args=()
        if [ -z "$manifest_ns" ]; then
          apply_args=(-n "$namespace")
        fi
        if ! kubectl apply "${apply_args[@]}" -f "$manifest_file"; then
          echo "ERROR: suite $suite_name failed to apply manifest $manifest_file" >&2
          exit 1
        fi
        applied_manifest_paths+=("$manifest_file")
      done
    fi

    if [ -n "$service_account" ]; then
      service_account_ready=0
      for _ in $(seq 1 30); do
        if kubectl get serviceaccount "$service_account" -n "$namespace" >/dev/null 2>&1; then
          service_account_ready=1
          break
        fi
        sleep 2
      done
      if [ "$service_account_ready" -ne 1 ]; then
        echo "ERROR: suite $suite_name service account $service_account not found in namespace $namespace" >&2
        exit 1
      fi
    fi

    pod_name="e2e-${suite_slug}-$(date +%s)"
    service_account_line=""
    if [ -n "$service_account" ]; then
      service_account_line="  serviceAccountName: $service_account"
    fi
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: ${pod_name}
  namespace: ${namespace}
  labels:
    app.kubernetes.io/managed-by: e2e
    app.kubernetes.io/name: ${suite_slug}
spec:
${service_account_line}
  restartPolicy: Never
  containers:
    - name: runner
      image: ${image}
      command:
        - sleep
        - infinity
EOF

    kubectl wait --for=condition=Ready "pod/${pod_name}" -n "$namespace" --timeout=300s
    kubectl exec -n "$namespace" "$pod_name" -- mkdir -p "$workdir"
    kubectl cp "$suite_dir/." "$namespace/$pod_name:$workdir"

    provider_binary_host="$suite_dir/.provider/terraform-provider-agyn"
    exec_env=()
    if [ -n "$tags" ]; then
      exec_env+=("TAGS=$tags")
    fi
    if [ -f "$provider_binary_host" ]; then
      exec_env+=("PROVIDER_BINARY=$workdir/.provider/terraform-provider-agyn")
    fi

    gateway_internal_url="http://gateway-gateway.${namespace}.svc.cluster.local:8080"
    if [ -n "${AGYN_BASE_URL:-}" ]; then
      exec_env+=("AGYN_BASE_URL=${AGYN_BASE_URL}")
    else
      exec_env+=("AGYN_BASE_URL=${gateway_internal_url}")
    fi

    for env_name in AGYN_MODEL_ID AGYN_AGENT_IMAGE AGYN_AGENT_INIT_IMAGE AGYN_API_TOKEN AGYN_ORGANIZATION_ID; do
      env_value=${!env_name:-}
      if [ -n "$env_value" ]; then
        exec_env+=("${env_name}=${env_value}")
      fi
    done

    for env_name in CODEX_INIT_IMAGE AGN_INIT_IMAGE CLAUDE_INIT_IMAGE AGN_EXPOSE_INIT_IMAGE; do
      env_value=${!env_name:-}
      if [ -n "$env_value" ]; then
        exec_env+=("${env_name}=${env_value}")
      fi
    done

    while IFS='=' read -r env_name env_value; do
      if [[ "$env_name" == E2E_* || "$env_name" == ARGOS_* ]]; then
        exec_env+=("${env_name}=${env_value}")
      fi
    done < <(env)

    exec_cmd=()
    if [ "${#exec_env[@]}" -gt 0 ]; then
      exec_cmd=(env "${exec_env[@]}")
    fi

    set +e
    kubectl exec -i -n "$namespace" "$pod_name" -- "${exec_cmd[@]}" bash -s < "$run_file"
    run_status=$?
    set -e

    suite_artifacts_dir="$artifacts_dir/$suite_name"
    mkdir -p "$suite_artifacts_dir"
    if kubectl exec -n "$namespace" "$pod_name" -- test -f "$workdir/junit.xml"; then
      if ! kubectl cp "$namespace/$pod_name:$workdir/junit.xml" "$suite_artifacts_dir/junit.xml"; then
        echo "ERROR: failed to copy junit.xml for suite $suite_name" >&2
        run_status=1
      fi
    else
      echo "ERROR: junit.xml missing for suite $suite_name" >&2
      run_status=1
    fi

    if [ "$run_status" -ne 0 ]; then
      suite_diagnostics_dir="$diagnostics_root/$suite_name"
      mkdir -p "$suite_diagnostics_dir/logs"
      kubectl get pod "$pod_name" -n "$namespace" -o wide > "$suite_diagnostics_dir/pod.txt" 2>&1 || true
      kubectl describe pod "$pod_name" -n "$namespace" > "$suite_diagnostics_dir/describe.txt" 2>&1 || true
      kubectl logs -n "$namespace" --all-containers --prefix "$pod_name" \
        > "$suite_diagnostics_dir/logs/${pod_name}.log" 2>&1 || true
      kubectl get events -n "$namespace" --sort-by=.metadata.creationTimestamp \
        > "$suite_diagnostics_dir/events.txt" 2>&1 || true
    fi

    exit "$run_status"
  ); then
    overall_exit=1
  fi
done

exit "$overall_exit"
