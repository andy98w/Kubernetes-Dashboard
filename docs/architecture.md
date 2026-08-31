# Architecture and scope

## Goal

Demonstrate production Kubernetes engineering on AWS through a system that can
be deployed, observed, secured, failed, and recovered. The repository favors
explainable engineering over a long list of logos.

## Target platform

- **Compute:** EKS managed control plane and managed node group; Karpenter is a
  later optimization after the baseline is measurable.
- **Networking:** AWS VPC CNI is the safest initial EKS default. Cilium runs in
  network-policy-only mode with Hubble visibility. Full CNI replacement is an
  advanced profile, not the default, because it changes bootstrap and support
  characteristics.
- **Ingress:** AWS Load Balancer Controller, ACM TLS, Route 53 DNS.
- **Identity:** EKS access entries for humans and EKS Pod Identity per workload.
- **Delivery:** GitHub Actions builds/scans/signs; Argo CD reconciles deployment.
- **Observability:** OpenTelemetry instrumentation and collectors, Prometheus,
  Grafana, Loki, and Tempo. SLOs and alerts live with the workload.
- **Security:** private worker subnets, KMS envelope encryption, Secrets Manager
  through External Secrets, Kyverno admission policy, Trivy, read-only RBAC,
  default-deny network policy, Pod Security Standards.

Each environment must narrow the dashboard chart's `networkPolicy.apiServerCidr`
from its portable bootstrap default to the private EKS API endpoint CIDR.

## Production profile versus portfolio profile

The portfolio profile uses one region, one cluster, and small on-demand nodes.
A real organization should use separate AWS accounts and clusters per lifecycle
boundary, multi-region recovery objectives where justified, central identity,
and a remote Terraform state backend with locking and recovery controls.

## Intentional exclusions

- The dashboard does not edit or delete Kubernetes resources.
- Service mesh is excluded until there is a concrete mTLS or traffic-management
  requirement; Cilium and standard telemetry cover this project's needs.
- Vault is excluded because AWS Secrets Manager plus External Secrets avoids
  operating another critical stateful control plane for a single-cloud demo.
- Running every CNCF tool is not a production practice. Each addition must map
  to a threat, SLO, operational constraint, or documented experiment.
