#!/usr/bin/env bash
set -euo pipefail

aws_profile="${AWS_PROFILE:-kubevista}"
aws_region="${AWS_REGION:-us-west-2}"
cluster_name="${KUBEVISTA_CLUSTER_NAME:-kubevista-dev}"
failures=0

if aws eks describe-cluster --profile "${aws_profile}" --region "${aws_region}" --name "${cluster_name}" >/dev/null 2>&1; then
  echo "FAIL: EKS cluster ${cluster_name} still exists."
  failures=$((failures + 1))
else
  echo "PASS: EKS cluster is absent."
fi

nat_count="$(aws ec2 describe-nat-gateways \
  --profile "${aws_profile}" \
  --region "${aws_region}" \
  --filter Name=tag:Project,Values=KubeVista Name=state,Values=pending,available,deleting \
  --query 'length(NatGateways)' --output text)"
if [[ "${nat_count}" != "0" ]]; then
  echo "FAIL: ${nat_count} tagged NAT gateway(s) remain."
  failures=$((failures + 1))
else
  echo "PASS: no tagged NAT gateways remain."
fi

leftover_count="$(aws resourcegroupstaggingapi get-resources \
  --profile "${aws_profile}" \
  --region "${aws_region}" \
  --tag-filters Key=Project,Values=KubeVista \
  --resource-type-filters ec2:volume elasticloadbalancing:loadbalancer \
  --query 'length(ResourceTagMappingList)' --output text)"
if [[ "${leftover_count}" != "0" ]]; then
  echo "FAIL: ${leftover_count} tagged EBS volume or load balancer resource(s) remain."
  failures=$((failures + 1))
else
  echo "PASS: no tagged EBS volumes or load balancers remain."
fi

if (( failures > 0 )); then
  echo "Cleanup verification failed. Inspect retained resources before considering teardown complete." >&2
  exit 1
fi

echo "Cleanup verification passed. The Terraform state bucket and its KMS key are intentionally retained."
