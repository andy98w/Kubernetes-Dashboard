# Architecture and scope

## Goal

I wanted one project that covered the full EKS lifecycle: provision it, deploy
an application, observe it under load, break part of it, recover, and clean up
the AWS resources afterward. The dashboard gives that platform a real workload
instead of leaving the repository as Terraform and YAML alone.

## Target platform

- **Compute:** EKS managed control plane and one managed node group.
- **Networking:** AWS VPC CNI with its NetworkPolicy agent. Cilium and Hubble
  are not installed; they remain a possible separate networking profile.
- **Ingress:** AWS Load Balancer Controller, ACM TLS, Route 53 DNS.
- **Identity:** EKS access entries for humans and EKS Pod Identity per workload.
- **Delivery:** GitHub Actions builds/scans/signs; Argo CD reconciles deployment.
- **Observability:** OpenTelemetry instrumentation and collectors, Prometheus,
  Grafana, Loki, and Tempo.
- **Security:** private worker subnets, KMS envelope encryption, Secrets Manager
  through External Secrets, Trivy scanning, read-only-by-default RBAC,
  opt-in namespace-scoped Deployment operations, NetworkPolicies,
  and restricted container security contexts. Kyverno is not installed.

The dashboard chart defaults `networkPolicy.apiServerCidr` to the exact
`kubernetes.default` Service ClusterIP used by the dev cluster. Each additional
environment must replace it with its own API Service IP (or the smallest
practical service CIDR); the managed control-plane VPC endpoint is not the
destination seen by in-cluster clients.

Prometheus, Loki, and Tempo use encrypted gp3 volumes. Loki and Tempo each run
as a single replica to keep the test environment affordable. For a long-running
environment I would move logs and traces to object storage, add replicas across
Availability Zones, and test restores.

## Test environment versus long-running production

This repository uses one region, one cluster, and a single NAT gateway. A
long-running production setup would normally separate environments into AWS
accounts, use one NAT gateway per Availability Zone, centralize audit logs, and
define recovery objectives before adding multi-region complexity.

## Guarded operation boundary

The original August 31 AWS run used read-only access. The later operator
workflow is implemented and testable in the static simulation, but has not been
claimed as exercised against that torn-down cluster. In a future live run,
`operations.enabled` must be reviewed explicitly. The browser cannot submit
arbitrary Kubernetes objects: it can request only a Deployment rollout restart
or a replica count inside the configured range, and the API performs a dry run
before the real request. Review plans are bound to the exact request and
resource version, expire after five minutes, and can be executed only once.

## Out of scope

- Arbitrary manifest editing, Secret access, RBAC changes, pod exec, controller
  deletion, and other unrestricted Kubernetes operations.
- Service mesh is excluded until there is a concrete mTLS or traffic-management
  requirement; NetworkPolicies and the existing telemetry cover this test.
- Vault is excluded because AWS Secrets Manager plus External Secrets avoids
  operating another critical stateful control plane for a single-cloud demo.
- Cilium/Hubble and Kyverno are roadmap experiments, not deployed components.
