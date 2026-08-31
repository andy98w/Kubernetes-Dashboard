# Deployment guide

The AWS infrastructure layer is the next implementation milestone. Until it is
complete, do not apply partial Terraform copied from the internet.

## Planned workflow

1. Authenticate with a short-lived AWS IAM Identity Center session.
2. Create the remote Terraform state bucket and lock table in a bootstrap stack.
3. Apply the `dev` VPC/EKS environment from CI with manual approval.
4. Bootstrap Argo CD once, then let the root application reconcile the platform.
5. Build, scan, generate an SBOM, sign the image, and update its immutable digest.
6. Run smoke, policy, load, and recovery checks and store evidence.

## Cost guardrails

EKS has a per-cluster hourly charge, and worker nodes, NAT gateways, load
balancers, logs, and metrics add cost. Use AWS Budgets, tag all resources, prefer
a single NAT gateway only in the portfolio profile, and destroy the environment
after demonstrations. Production would use a NAT gateway per availability zone.

## Required local tools

Pinned versions are documented in `.tool-versions`. Use `asdf`, `mise`, or your
preferred version manager; CI is the source of truth for repeatable validation.
