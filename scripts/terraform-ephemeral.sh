#!/usr/bin/env bash
set -euo pipefail

operation="${1:-}"
ttl_hours="${KUBEVISTA_TTL_HOURS:-8}"
aws_profile="${AWS_PROFILE:-kubevista}"
expected_account="${KUBEVISTA_AWS_ACCOUNT_ID:-}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
terraform_root="${repo_root}/infra/terraform/environments/dev"
plan_file="${terraform_root}/dev.tfplan"
destroy_plan_file="${terraform_root}/destroy.tfplan"

usage() {
  echo "Usage: AWS_PROFILE=kubevista KUBEVISTA_AWS_ACCOUNT_ID=<account> $0 plan|apply|destroy"
  echo "Optional: KUBEVISTA_TTL_HOURS=4 (default: 8)"
}

if [[ ! "${operation}" =~ ^(plan|apply|destroy)$ ]]; then
  usage
  exit 2
fi

if [[ -z "${expected_account}" ]]; then
  echo "KUBEVISTA_AWS_ACCOUNT_ID is required to guard against using the wrong AWS account." >&2
  exit 2
fi

for command in aws terraform python3; do
  command -v "${command}" >/dev/null || { echo "Missing required command: ${command}" >&2; exit 2; }
done

actual_account="$(aws sts get-caller-identity --profile "${aws_profile}" --query Account --output text)"
if [[ "${actual_account}" != "${expected_account}" ]]; then
  echo "Refusing to continue: authenticated account ${actual_account} != expected ${expected_account}." >&2
  exit 1
fi

terraform_args=(-chdir="${terraform_root}")
export AWS_PROFILE="${aws_profile}"

terraform "${terraform_args[@]}" init -reconfigure -input=false -backend-config=backend.hcl

case "${operation}" in
  plan)
    if ! [[ "${ttl_hours}" =~ ^([1-9]|1[0-9]|2[0-4])$ ]]; then
      echo "KUBEVISTA_TTL_HOURS must be an integer from 1 through 24." >&2
      exit 2
    fi
    export TF_VAR_expires_at
    TF_VAR_expires_at="$(python3 - "${ttl_hours}" <<'PY'
from datetime import datetime, timedelta, timezone
import sys
print((datetime.now(timezone.utc) + timedelta(hours=int(sys.argv[1]))).strftime("%Y-%m-%dT%H:%M:%SZ"))
PY
)"
    terraform "${terraform_args[@]}" plan -input=false -out="${plan_file}"
    mkdir -p "${repo_root}/work"
    printf 'expires_at=%s\n' "${TF_VAR_expires_at}" > "${repo_root}/work/ephemeral-deployment.env"
    echo "Plan saved. Teardown deadline: ${TF_VAR_expires_at}"
    ;;
  apply)
    [[ -f "${plan_file}" ]] || { echo "Run '$0 plan' first." >&2; exit 1; }
    read -r -p "Type APPLY kubevista-dev to continue: " confirmation
    [[ "${confirmation}" == "APPLY kubevista-dev" ]] || { echo "Confirmation did not match; aborting." >&2; exit 1; }
    terraform "${terraform_args[@]}" apply -input=false "${plan_file}"
    echo "Cluster applied. Destroy it by the recorded ExpiresAt deadline."
    ;;
  destroy)
    terraform "${terraform_args[@]}" plan -destroy -input=false -out="${destroy_plan_file}"
    terraform "${terraform_args[@]}" show -no-color "${destroy_plan_file}"
    read -r -p "Type DESTROY kubevista-dev to apply this destroy plan: " confirmation
    [[ "${confirmation}" == "DESTROY kubevista-dev" ]] || { echo "Confirmation did not match; aborting." >&2; exit 1; }
    terraform "${terraform_args[@]}" apply -input=false "${destroy_plan_file}"
    "${repo_root}/scripts/verify-aws-cleanup.sh"
    ;;
esac
