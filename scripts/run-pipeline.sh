#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
parse_script="$repo_root/scripts/parse_suite.py"
suites_dir="$repo_root/suites"
artifacts_dir="$repo_root/.artifacts/junit"
diagnostics_root="$repo_root/.diagnostics/suites"

tags="${TAGS:-}"
namespace="${E2E_NAMESPACE:-${DEVSPACE_NAMESPACE:-platform}}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl not found in PATH" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker not found in PATH" >&2
  exit 1
fi

if [ ! -d "$suites_dir" ]; then
  echo "ERROR: suites directory not found at $suites_dir" >&2
  exit 1
fi

rm -rf "$artifacts_dir" "$diagnostics_root"
mkdir -p "$artifacts_dir" "$diagnostics_root"

if ! kubectl get namespace "$namespace" >/dev/null 2>&1; then
  kubectl create namespace "$namespace"
fi

mapfile -t suite_files < <(find "$suites_dir" -mindepth 2 -maxdepth 2 -name suite.yaml | sort)

if [ "${#suite_files[@]}" -eq 0 ]; then
  echo "ERROR: No suites found under $suites_dir" >&2
  exit 1
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

    cleanup() {
      local exit_code=$?
      set +e
      if [ -n "${pod_name:-}" ]; then
        kubectl delete pod "$pod_name" -n "$namespace" --wait=true --ignore-not-found
      fi
      if [ "${#applied_manifest_paths[@]}" -gt 0 ]; then
        for manifest_path in "${applied_manifest_paths[@]}"; do
          kubectl delete -n "$namespace" -f "$manifest_path" --ignore-not-found
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

    service_account=""
    if [ -s "$service_account_file" ]; then
      service_account=$(cat "$service_account_file" | xargs)
    fi

    manifests=()
    if [ -s "$manifests_file" ]; then
      while IFS= read -r manifest; do
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

    if ! select_output=$(docker run --rm -i -e TAGS="$tags" -v "$suite_dir":"$workdir" -w "$workdir" "$image" bash -s < "$select_file"); then
      echo "ERROR: select failed for suite $suite_name" >&2
      exit 1
    fi

    if [ -z "$(echo "$select_output" | tr -d '[:space:]')" ]; then
      echo "Skipping suite $suite_name (no matching tests)."
      exit 0
    fi

    manifest_paths=()
    if [ "${#manifests[@]}" -gt 0 ]; then
      for manifest in "${manifests[@]}"; do
        manifest_path="$suite_dir/$manifest"
        if [ ! -e "$manifest_path" ]; then
          echo "ERROR: suite $suite_name manifest not found at $manifest_path" >&2
          exit 1
        fi
        manifest_paths+=("$manifest_path")
      done
      for manifest_path in "${manifest_paths[@]}"; do
        if ! kubectl apply -n "$namespace" -f "$manifest_path"; then
          echo "ERROR: suite $suite_name failed to apply manifest $manifest_path" >&2
          exit 1
        fi
        applied_manifest_paths+=("$manifest_path")
      done
    fi

    pod_name="e2e-${suite_slug}-$(date +%s)"
    service_account_args=()
    if [ -n "$service_account" ]; then
      service_account_args=("--serviceaccount=$service_account")
    fi
    kubectl run "$pod_name" -n "$namespace" \
      --image="$image" \
      --restart=Never \
      --labels="app.kubernetes.io/managed-by=e2e,app.kubernetes.io/name=${suite_slug}" \
      "${service_account_args[@]}" \
      --command -- sleep infinity

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

    for env_name in CODEX_INIT_IMAGE AGN_INIT_IMAGE AGN_EXPOSE_INIT_IMAGE; do
      env_value=${!env_name:-}
      if [ -n "$env_value" ]; then
        exec_env+=("${env_name}=${env_value}")
      fi
    done

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
