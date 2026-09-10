# Test records

The AWS environment ran long enough to deploy and test the complete stack, then
was removed to stop the hourly charges. These notes contain the measurements
and AWS/Kubernetes checks collected during that run.

| Area | Record | What was checked |
| --- | --- | --- |
| AWS foundation | [Live EKS deployment](live-eks-2026-08-31.md#what-was-deployed) | Terraform created 92 resources across the VPC, EKS, IAM, encryption, logging, and budget layers |
| GitOps-managed platform | [GitOps and observability result](live-eks-2026-08-31.md#gitops-and-observability) | 12 Argo CD applications reached `Synced` and `Healthy` |
| Signed immutable supply chain | [Signed release](live-eks-2026-08-31.md#signed-release) | GitHub OIDC release, digest deployment, SBOM/provenance, and Sigstore referrers |
| Availability during disruption | [Load and recovery tests](live-eks-2026-08-31.md#load-and-recovery-tests) | 896/896 successful requests during both API and web pod replacement |
| Least-privilege runtime | [Security checks](live-eks-2026-08-31.md#security-and-policy-checks) | Positive and negative RBAC and NetworkPolicy tests |
| Authenticated public delivery | [Public delivery verification](live-eks-2026-08-31.md#authenticated-public-delivery) | TLS, Cognito redirect, healthy multi-AZ targets, DNS, and no-drift plan |
| Cost-controlled lifecycle | [Verified teardown](teardown-2026-08-31.md) | Direct AWS service APIs confirmed billable infrastructure was absent |
| Disruption test | [Controlled disruption case study](controlled-disruption-case-study.md) | Traffic, replacement time, response count, and test limits |

## Limits of the test environment

- The public demo uses representative data because the AWS environment is
  offline.
- The load tests covered one pod replacement at a time. They did not measure
  maximum capacity or long-term stability.
- Loki and Tempo used one replica each with short retention. A long-running
  installation would need object storage, more replicas, and restore tests.
- Cost figures are directional unless verified against an AWS invoice or
  Cost Explorer export.
