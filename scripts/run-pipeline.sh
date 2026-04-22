#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
parse_script="$repo_root/scripts/parse_suite.py"
suites_dir="$repo_root/suites"
artifacts_dir="$repo_root/.artifacts/junit"

tags="${TAGS:-}"
namespace="${DEVSPACE_NAMESPACE:-${E2E_NAMESPACE:-platform}}"

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

rm -rf "$artifacts_dir"
mkdir -p "$artifacts_dir"

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
  suite_dir=$(dirname "$suite_file")
  suite_name=$(basename "$suite_dir")
  suite_slug=$(echo "$suite_name" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-' | sed 's/^-//;s/-$//')
  tmp_dir=$(mktemp -d)

  python3 "$parse_script" "$suite_file" "$tmp_dir"
  image=$(cat "$tmp_dir/image")
  workdir=$(cat "$tmp_dir/workdir")
  select_file="$tmp_dir/select"
  run_file="$tmp_dir/run"

  if [ -z "$image" ]; then
    echo "ERROR: suite $suite_name missing image" >&2
    overall_exit=1
    rm -rf "$tmp_dir"
    continue
  fi

  if [ -z "$workdir" ]; then
    workdir="/opt/app/data"
  fi

  if [ ! -s "$select_file" ]; then
    echo "ERROR: suite $suite_name missing select command" >&2
    overall_exit=1
    rm -rf "$tmp_dir"
    continue
  fi

  if [ ! -s "$run_file" ]; then
    echo "ERROR: suite $suite_name missing run command" >&2
    overall_exit=1
    rm -rf "$tmp_dir"
    continue
  fi

  if ! select_output=$(docker run --rm -i -e TAGS="$tags" -v "$suite_dir":"$workdir" -w "$workdir" "$image" bash -s < "$select_file"); then
    echo "ERROR: select failed for suite $suite_name" >&2
    overall_exit=1
    rm -rf "$tmp_dir"
    continue
  fi

  if [ -z "$(echo "$select_output" | tr -d '[:space:]')" ]; then
    echo "Skipping suite $suite_name (no matching tests)."
    rm -rf "$tmp_dir"
    continue
  fi

  pod_name="e2e-${suite_slug}-$(date +%s)"
  kubectl run "$pod_name" -n "$namespace" \
    --image="$image" \
    --restart=Never \
    --labels="app.kubernetes.io/managed-by=e2e,app.kubernetes.io/name=${suite_slug}" \
    --command -- sleep infinity

  kubectl wait --for=condition=Ready "pod/${pod_name}" -n "$namespace" --timeout=300s
  kubectl exec -n "$namespace" "$pod_name" -- mkdir -p "$workdir"
  kubectl cp "$suite_dir/." "$namespace/$pod_name:$workdir"

  provider_binary_host="$suite_dir/.provider/terraform-provider-agyn"
  provider_env=()
  if [ -f "$provider_binary_host" ]; then
    provider_env=(env PROVIDER_BINARY="$workdir/.provider/terraform-provider-agyn")
  fi

  set +e
  kubectl exec -i -n "$namespace" "$pod_name" -- "${provider_env[@]}" bash -s < "$run_file"
  run_status=$?
  set -e

  suite_artifacts_dir="$artifacts_dir/$suite_name"
  mkdir -p "$suite_artifacts_dir"
  if kubectl exec -n "$namespace" "$pod_name" -- test -f "$workdir/junit.xml"; then
    kubectl cp "$namespace/$pod_name:$workdir/junit.xml" "$suite_artifacts_dir/junit.xml"
  else
    echo "ERROR: junit.xml missing for suite $suite_name" >&2
    run_status=1
  fi

  if ! kubectl delete pod "$pod_name" -n "$namespace" --wait=true; then
    echo "ERROR: Failed to delete pod $pod_name" >&2
    run_status=1
  fi

  if [ "$run_status" -ne 0 ]; then
    overall_exit=1
  fi

  rm -rf "$tmp_dir"
done

exit "$overall_exit"
