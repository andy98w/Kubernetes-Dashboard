# KubeVista

KubeVista is a production-minded Kubernetes operations dashboard and AWS EKS
reference platform. It is designed as a portfolio project: every component has
a reason to exist, a documented trade-off, and an automated verification path.

## What this demonstrates

- A Go control-plane API using Kubernetes `client-go` and least-privilege RBAC
- A React/TypeScript dashboard for workload and cluster health
- Terraform-managed three-tier VPC and Amazon EKS infrastructure
- GitOps delivery with Argo CD and Helm
- AWS VPC CNI, EBS CSI, EKS Pod Identity, External Secrets, and AWS Load Balancer Controller
- Prometheus, Grafana, Loki, Tempo, and OpenTelemetry metrics, logs, and traces
- CI checks, autoscaling, disruption budgets, probes, and graceful shutdown
- Immutable ECR images with SBOM/provenance attestations and keyless Cosign signatures

## Architecture

```text
Browser -> ALB/HTTPS -> dashboard web -> Go API -> Kubernetes API
                            |              |
                            |              +-> OpenTelemetry -> Tempo
                            +-> metrics/logs -> Prometheus/Loki -> Grafana

GitHub Actions -> ECR -> Argo CD -> EKS
Terraform --------------------^     |-- VPC CNI + EBS CSI
                                    |-- Pod Identity controllers
                                    |-- External Secrets
                                    `-- OTel + Prometheus stack
```

The dashboard is intentionally **read-only by default**. Mutating cluster tools
look impressive in demos but create an unnecessarily dangerous security model.

## Repository layout

| Path | Purpose |
| --- | --- |
| `api/` | Go API and Kubernetes adapter |
| `web/` | React/TypeScript UI |
| `infra/terraform/` | AWS network, EKS, IAM, and add-ons |
| `platform/` | Argo CD and Helm definitions |
| `docs/` | Architecture decisions, runbooks, and threat model |

The complete eight-layer inventory is in
[docs/platform-stack.md](docs/platform-stack.md), and operational commands are
in [docs/runbook.md](docs/runbook.md). Upstream design sources are collected in
[docs/references.md](docs/references.md).

The frontend deliberately avoids a generic component-library look. Its visual
and data-integrity rules are documented in [docs/frontend.md](docs/frontend.md),
and the container trust path is documented in
[docs/supply-chain.md](docs/supply-chain.md).

## Local development

Prerequisites: Go 1.26+, Node 24+, Docker, kubectl, Helm, and optionally kind.

```bash
make test
make run-api
make run-web
```

The API uses your current kubeconfig outside a cluster and in-cluster service
account credentials on EKS. Set `KUBEVISTA_DEMO_MODE=true` to run without a
cluster while developing the UI.

## AWS deployment

Start with [docs/deployment.md](docs/deployment.md). A continuously running EKS
environment costs real money; use the documented teardown workflow when the
demo is not needed. Never commit Terraform state or AWS credentials.

## Roadmap

- [x] Repository architecture and production guardrails
- [x] Go health/readiness API and container image
- [x] Live Kubernetes node, namespace, and pod inventory API
- [x] Helm workload with RBAC, PDB, autoscaling, and network policy
- [x] Terraform VPC/EKS baseline, encrypted remote-state bootstrap, and cost budget
- [x] Pod Identity-backed AWS integrations and encrypted EBS storage
- [x] GitOps definitions for Prometheus, Grafana, Loki, Tempo, and OpenTelemetry
- [ ] Kubernetes event and workload detail APIs
- [ ] Workload drill-down UI and Hubble flows
- [x] GitOps platform add-on bootstrap definitions
- [x] Deployable web tier and signed ECR image supply chain
- [ ] Cilium/Hubble advanced networking profile
- [ ] Kyverno signature enforcement and runtime security profile
- [ ] Load, failure, recovery, and security evidence in `docs/evidence/`

See [docs/architecture.md](docs/architecture.md) for scope and engineering decisions.
