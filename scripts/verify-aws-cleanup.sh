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

volume_count="$(aws ec2 describe-volumes \
  --profile "${aws_profile}" \
  --region "${aws_region}" \
  --filters Name=tag:Project,Values=KubeVista \
  --query 'length(Volumes)' --output text)"

load_balancer_count=0
while IFS= read -r load_balancer_arn; do
  [[ -z "${load_balancer_arn}" ]] && continue
  project_tag_count="$(aws elbv2 describe-tags \
    --profile "${aws_profile}" \
    --region "${aws_region}" \
    --resource-arns "${load_balancer_arn}" \
    --query 'length(TagDescriptions[0].Tags[?Key == `Project` && Value == `KubeVista`])' \
    --output text)"
  if [[ "${project_tag_count}" != "0" ]]; then
    load_balancer_count=$((load_balancer_count + 1))
  fi
done < <(aws elbv2 describe-load-balancers \
  --profile "${aws_profile}" \
  --region "${aws_region}" \
  --query 'LoadBalancers[].LoadBalancerArn' --output text | tr '\t' '\n')

if [[ "${volume_count}" != "0" || "${load_balancer_count}" != "0" ]]; then
  echo "FAIL: ${volume_count} tagged EBS volume(s) and ${load_balancer_count} tagged load balancer(s) remain."
  failures=$((failures + 1))
else
  echo "PASS: no tagged EBS volumes or load balancers remain."
fi

if (( failures > 0 )); then
  echo "Cleanup verification failed. Inspect retained resources before considering teardown complete." >&2
  exit 1
fi

echo "Cleanup verification passed. The Terraform state bucket and its KMS key are intentionally retained."
