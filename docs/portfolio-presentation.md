# Portfolio presentation guide

## Résumé bullets

Choose three or four based on the role. Every number below is tied to the
[evidence index](evidence/README.md).

- Designed and deployed a production-oriented Kubernetes platform on AWS EKS
  using Terraform, a three-tier multi-AZ VPC, private worker nodes, KMS
  encryption, IAM Identity Center, EKS Pod Identity, and least-privilege RBAC.
- Built KubeVista, a Go, `client-go`, React, and TypeScript operations console
  that correlates controllers with Pods, images, Services, NetworkPolicies,
  Events, security findings, and directional infrastructure cost.
- Added an opt-in Deployment restart and scaling path with namespace and replica
  guardrails, Kubernetes server-side dry runs, five-minute single-use plans,
  resource-version checks, verified ALB identity, and audit receipts; inventory
  remains read-only by default.
- Implemented GitOps delivery with Argo CD and Helm for 12 platform
  applications, including autoscaling, disruption budgets, topology spreading,
  health probes, encrypted persistent storage, and network isolation.
- Established an OpenTelemetry observability pipeline with Prometheus, Grafana,
  Loki, and Tempo for metrics, logs, traces, alerting, and application telemetry.
- Secured image delivery with GitHub OIDC, immutable ECR digests, SBOM and
  provenance attestations, vulnerability scanning, and keyless Cosign
  signatures.
- Validated single-pod disruption tolerance at 20 requests/second: replacement
  API and web Pods became Ready in 2 seconds while `896/896` requests to each
  tier returned HTTP 200.
- Reduced ongoing portfolio cost by separating the ephemeral, fully validated
  EKS environment from a static recruiter demo and verifying destruction through
  direct AWS service APIs.

Recommended set for a platform or DevOps role: bullets 1, 3, 4, and 7.
Recommended set for a software or full-stack role: bullets 2, 3, 5, and 7.

## 90-second recruiter walkthrough

1. Open **Overview** and explain that the public build is representative data
   from the intentionally destroyed EKS environment, with proof linked from the
   banner.
2. Open **Workloads**, select `kubevista-api`, and show both the correlated
   workload evidence and the guarded restart/scale review flow. Be explicit that
   the public build simulates operations and the original AWS run was read-only.
3. Open **Incidents** and show the controlled pod-loss timeline: 20 requests per
   second, two-second replacement, and `896/896` HTTP 200 responses.
4. Open **Observability** and explain the metrics/logs/traces path; note that
   Grafana and Argo CD were private administrative surfaces.
5. Finish with the GitHub evidence index, signed image digests, and verified
   teardown record.

## Interview story: resilience defect to verified behavior

**Situation:** Repository-only validation could not expose every interaction
between distroless containers, NetworkPolicies, VPC CNI enforcement, and the
deployed observability stack.

**Task:** Bring the platform to a reproducible, secure state and prove that the
application stayed available during a realistic disruption.

**Action:** Corrected five live integration defects, enabled VPC CNI policy
enforcement, constrained Kubernetes API egress, added positive and negative
policy tests, promoted signed digest-pinned images, and removed one API and one
web Pod while Fortio generated continuous traffic.

**Result:** All 12 Argo CD applications converged, both replacements became
Ready in two seconds, all `896/896` requests per disruption test succeeded, and
direct AWS service queries later confirmed that billable infrastructure was
destroyed.

## Claim boundaries

Say “production-oriented” or “production-minded,” not “served production
customers.” The test establishes continuity during one controlled Pod loss, not
an Availability Zone failure or a capacity limit. The public demo is
representative and explicitly labeled; the live EKS deployment and its teardown
are independently documented.
