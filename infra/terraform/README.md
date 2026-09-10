# AWS infrastructure

There are two Terraform root modules:

- `bootstrap/` creates the encrypted, versioned S3 state bucket and KMS key.
- `environments/dev/` creates the VPC, EKS cluster, nodes, logs,
  access entry, managed add-ons, Pod Identity roles, and cost budget.

Terraform and provider/module versions are constrained and their dependency
locks are committed. CI runs formatting, initialization, and static validation
without AWS credentials. Plans and applies remain explicit operator actions.

## Bootstrap state

```bash
cd infra/terraform/bootstrap
cp terraform.tfvars.example terraform.tfvars
terraform init -backend=false
terraform plan -out=bootstrap.tfplan
terraform apply bootstrap.tfplan
```

The bucket name must be globally unique. Save the outputs, then migrate the
bootstrap state into the bucket it created:

```bash
terraform init -migrate-state -force-copy \
  -backend-config="bucket=<state_bucket_name>" \
  -backend-config="key=kubevista/bootstrap/terraform.tfstate" \
  -backend-config="region=us-west-2" \
  -backend-config="encrypt=true" \
  -backend-config="kms_key_id=<state_kms_key_arn>" \
  -backend-config="use_lockfile=true"
```

The backend uses native S3 lock files (`use_lockfile`), so no DynamoDB lock
table is required. Configure the environment backend only after this migration
succeeds.

## Plan the dev environment

Copy the ignored backend and variable files and replace their placeholders:

```bash
cd infra/terraform/environments/dev
cp backend.hcl.example backend.hcl
cp terraform.tfvars.example terraform.tfvars
terraform init -reconfigure -backend-config=backend.hcl
terraform plan -out=dev.tfplan
terraform show dev.tfplan
terraform apply dev.tfplan
```

`backend.hcl` and `terraform.tfvars` are ignored because they contain
account-specific operator settings. The committed example files remain the
reviewable source for every required value.

The default API endpoint is private-only. A local workstation can reach it only
through private connectivity. A temporary demo may set `public_access_cidrs` to
the operator's exact public `/32`. The example rejects broad public CIDRs.

The EKS add-on lifecycle owns the Pod Identity associations for VPC CNI and EBS
CSI. Terraform separately associates service accounts for AWS Load Balancer
Controller and External Secrets. External Secrets can read only Secrets Manager
resources under `external_secret_prefix`; no AWS access keys are stored in
Kubernetes.

## Differences for a long-running environment

This configuration uses one NAT gateway to control cost. Set
`single_nat_gateway = false` for one NAT gateway per availability zone. A real
production organization should also use separate AWS accounts, a private CI
runner, protected plan/apply environments, centralized audit logs, backup and
restore testing, and workload-specific IAM through EKS Pod Identity.

Destroy the EKS environment after demonstrations to stop most recurring costs.
Use the controller-aware sequence and post-destroy checks in
[`docs/runbook.md`](../../docs/runbook.md); a raw Terraform destroy does not own
ExternalDNS records or dynamically provisioned PVC volumes.

The underlying saved-plan workflow is:

```bash
terraform plan -destroy -out=destroy.tfplan
terraform apply destroy.tfplan
```

The remote-state bucket has `prevent_destroy`. Keep it for state recovery, or
remove that guard only after deciding how the state will be retained.
