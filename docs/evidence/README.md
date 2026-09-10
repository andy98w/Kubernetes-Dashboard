# KubeVista evidence index

This index maps recruiter-facing claims to durable repository evidence. The AWS
environment was intentionally ephemeral; retained documents record what was
observed while it was live and what was verified after destruction.

| Claim | Evidence | Verification |
| --- | --- | --- |
| Production-oriented AWS foundation | [Live EKS deployment](live-eks-2026-08-31.md#provisioning-result) | Terraform created 92 resources across the VPC, EKS, IAM, encryption, logging, and budget layers |
| GitOps-managed platform | [GitOps and observability result](live-eks-2026-08-31.md#gitops-and-observability-result) | 12 Argo CD applications reached `Synced` and `Healthy` |
| Signed immutable supply chain | [Signed release](live-eks-2026-08-31.md#signed-release) | GitHub OIDC release, digest deployment, SBOM/provenance, and Sigstore referrers |
| Availability during disruption | [Load and recovery evidence](live-eks-2026-08-31.md#load-and-recovery-evidence) | 896/896 successful requests during both API and web pod replacement |
| Least-privilege runtime | [Security assertions](live-eks-2026-08-31.md#security-and-policy-assertions) | Positive and negative RBAC and NetworkPolicy tests |
| Authenticated public delivery | [Public delivery verification](live-eks-2026-08-31.md#authenticated-public-delivery) | TLS, Cognito redirect, healthy multi-AZ targets, DNS, and no-drift plan |
| Cost-controlled lifecycle | [Verified teardown](teardown-2026-08-31.md) | Direct AWS service APIs confirmed billable infrastructure was absent |
| Incident-response thinking | [Controlled disruption case study](controlled-disruption-case-study.md) | Signal, impact, recovery, validation, and follow-up captured as one narrative |

## What remains intentionally different from a long-running production service

- The public recruiter build uses clearly labeled representative data because
  the AWS environment is intentionally offline.
- Load tests prove continuity for a single-pod disruption; they are not a
  capacity limit or long-duration soak test.
- The portfolio observability profile uses single-replica Loki and Tempo with
  short retention. The documented production profile moves durable telemetry
  to object storage and adds multi-AZ replicas.
- Cost figures are directional unless verified against an AWS invoice or
  Cost Explorer export.
