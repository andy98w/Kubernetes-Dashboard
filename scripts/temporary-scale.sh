#!/usr/bin/env bash
set -euo pipefail
# Foreground runbook: loss of this process prevents automatic restoration.
api="${KUBEVISTA_API:-http://127.0.0.1:8080}"
namespace="${1:?namespace required}"; name="${2:?deployment required}"
replicas="${3:?replicas required}"; duration="${4:-30}"
[[ "$duration" =~ ^[0-9]+$ && "$duration" -gt 0 && "$duration" -le 600 ]] || { echo "Duration must be 1–600 seconds" >&2;exit 1; }
[[ "$replicas" =~ ^[0-9]+$ ]] || exit 1
scratch=$(mktemp -d)
cli=(bin/kubevista)
base=(--api "$api" --namespace "$namespace" --name "$name")
"${cli[@]}" diagnose "${base[@]}" >"$scratch/before.json"
before=$(jq -r .workload.desired "$scratch/before.json")
"${cli[@]}" plan "${base[@]}" --action scale --replicas "$replicas" --reason "Temporary capacity for ${duration}s; restore to ${before}" >"$scratch/scale.json"
# Invocation explicitly requests both changes; plans are kept for inspection.
"${cli[@]}" execute --api "$api" --plan "$scratch/scale.json"
"${cli[@]}" diagnose "${base[@]}" >"$scratch/scaled.json"
generation=$(jq -r .recovery.generation "$scratch/scaled.json")
uid=$(jq -r .uid "$scratch/scaled.json")
restore() {
  trap - EXIT INT TERM
  "${cli[@]}" diagnose "${base[@]}" >"$scratch/current.json" || { echo "Cannot restore; inspect $scratch and restore manually" >&2;return 1; }
  if ! jq -e --arg uid "$uid" --argjson generation "$generation" --argjson replicas "$replicas" '.uid==$uid and .recovery.generation==$generation and .workload.desired==$replicas' "$scratch/current.json" >/dev/null; then
    echo "Deployment changed during temporary scale; restoration skipped. Review $scratch" >&2;return 1
  fi
  "${cli[@]}" plan "${base[@]}" --action scale --replicas "$before" --reason "Restore capacity after temporary scale" >"$scratch/restore.json"
  "${cli[@]}" execute --api "$api" --plan "$scratch/restore.json"
}
trap restore EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
sleep "$duration"
