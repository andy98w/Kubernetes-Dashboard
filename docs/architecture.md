# Architecture and scope

## Goal

Demonstrate production Kubernetes engineering on AWS through a system that can
be deployed, observed, secured, failed, and recovered. The repository favors
explainable engineering over a long list of logos.

## Target platform

- **Compute:** EKS managed control plane and managed node group; Karpenter is a
  later optimization after the baseline is measurable.
- **Networking:** AWS VPC CNI is the safest initial EKS default. Cilium runs in
  the baseline. Cilium/Hubble remains an advanced profile rather than being
  installed beside VPC CNI without an explicit chaining and support decision.
- **Ingress:** AWS Load Balancer Controller, ACM TLS, Route 53 DNS.
- **Identity:** EKS access entries for humans and EKS Pod Identity per workload.
- **Delivery:** GitHub Actions builds/scans/signs; Argo CD reconciles deployment.
- **Observability:** OpenTelemetry instrumentation and collectors, Prometheus,
  Grafana, Loki, and Tempo. SLOs and alerts live with the workload.
- **Security:** private worker subnets, KMS envelope encryption, Secrets Manager
  through External Secrets, Kyverno admission policy, Trivy, read-only RBAC,
  default-deny network policy, Pod Security Standards.

The dashboard chart defaults `networkPolicy.apiServerCidr` to the exact
`kubernetes.default` Service ClusterIP used by the dev cluster. Each additional
environment must replace it with its own API Service IP (or the smallest
practical service CIDR); the managed control-plane VPC endpoint is not the
destination seen by in-cluster clients.

The observability deployment is intentionally sized for a portfolio cluster:
Prometheus, Loki, and Tempo retain data on encrypted gp3 volumes, but Loki and
Tempo run as single replicas. A production profile should use object storage,
multi-AZ replicas, tested restore procedures, and retention based on SLO and
compliance requirements.

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
