# Deployment guide

The repository now contains a validated AWS infrastructure baseline. It creates
a three-AZ VPC, EKS control plane, managed node group, control-plane and VPC flow
logs, native EKS add-ons, KMS envelope encryption, access entries, and an AWS
Budget. Nothing is applied automatically by CI.

## Safe deployment workflow

1. Authenticate with a short-lived AWS IAM Identity Center session.
2. Copy `infra/terraform/bootstrap/terraform.tfvars.example` to an untracked
   `terraform.tfvars`, choose a globally unique bucket name, and apply the
   bootstrap stack. The bucket uses versioning, KMS encryption, TLS-only access,
   and native S3 state locking. This stack also creates the two persistent ECR
   repositories and the narrow GitHub image-publisher role.
3. Copy the bootstrap output into
   `infra/terraform/environments/dev/backend.tf.example`, save it as
   `backend.tf`, then copy `terraform.tfvars.example` to `terraform.tfvars`.
4. Set `admin_principal_arn` to a role, never an IAM user. Keep the API private
   when using a VPN or VPC runner; for a short portfolio demo, allow only your
   current public `/32` in `public_access_cidrs`.
5. Run `terraform plan -out=dev.tfplan`, inspect the complete plan, then apply
   that saved plan. Creating the cluster incurs AWS charges.
6. Configure kubectl with the `configure_kubectl` Terraform output. Bootstrap
   Argo CD once, then let the root application reconcile platform components.
7. Run smoke, policy, load, and recovery checks and store evidence.

Bootstrap the pinned Argo CD release after the cluster is reachable:

```bash
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update argo
helm upgrade --install argocd argo/argo-cd \
  --version 10.4.2 \
  --namespace argocd \
  --create-namespace \
  --values platform/argocd/values.yaml
kubectl apply -f platform/argocd/root-application.yaml
```

The root application creates a restricted Argo CD project and child
applications in sync waves: AWS controllers first, storage configuration next,
observability after that, and KubeVista last. See
[`docs/runbook.md`](runbook.md) for verification and troubleshooting commands.

Run the repository-only checks without AWS credentials:

```bash
make terraform-check
make platform-check
```

The detailed commands and design boundaries are in
[`infra/terraform/README.md`](../infra/terraform/README.md).

## Application image publication

The `Build and sign application images` workflow runs manually or for a `v*`
tag. Before running it, create a protected GitHub Environment named
`kubevista-images` with `AWS_ACCOUNT_ID`, `AWS_REGION`, and
`AWS_IMAGE_PUBLISHER_ROLE_ARN` environment variables. Copy the role ARN from
the `github_image_publisher_role_arn` bootstrap output. The role trust policy
accepts only this repository's durable owner and repository IDs, this
environment, and the STS audience. Durable IDs prevent a renamed or later
re-created repository from inheriting the trust relationship.

Each API and web build is pushed as `sha-<full commit>`. ECR rejects tag
replacement, scans the image on push, expires untagged layers after seven days,
and keeps the latest 20 application tags. BuildKit publishes an SBOM and SLSA
provenance attestation; Cosign signs the immutable digest with GitHub OIDC.
Deploy the digest reported in the workflow summary, not a mutable tag. See
[`docs/supply-chain.md`](supply-chain.md) for verification commands.

## Cost guardrails

EKS has a per-cluster hourly charge, and worker nodes, NAT gateways, load
balancers, logs, and metrics add cost. Use AWS Budgets, tag all resources, prefer
a single NAT gateway only in the portfolio profile, and destroy the environment
after demonstrations. Production would use a NAT gateway per availability zone.
The state bucket is protected by `prevent_destroy`, so retain it for audit and
recovery or remove that guard only through a deliberate state-retention process.
ECR storage is much cheaper than a running cluster but is not free; the lifecycle
policy bounds accumulated layers.

## Required local tools

Pinned versions are documented in `.tool-versions`. Use `asdf`, `mise`, or your
preferred version manager; CI is the source of truth for repeatable validation.
