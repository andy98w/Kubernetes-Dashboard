# Eight-layer AWS and Kubernetes platform

This document is the implementation inventory for KubeVista. It separates
resources by dependency order so a reviewer can see what creates each resource,
why it exists, and which trade-offs are portfolio-specific.

```text
1 Network -> 2 IAM -> 3 EKS -> 4 Compute -> 5 Core add-ons
                                              |
                                              v
8 Observability <- 7 Applications <- 6 AWS/GitOps integrations
```

## 1. AWS foundation

Terraform root: `infra/terraform/environments/dev`

| Resource | Implementation | Purpose |
| --- | --- | --- |
| VPC | `10.42.0.0/16`, DNS enabled | Isolated address and routing boundary |
| Public subnets | Three `/20` subnets across available AZs | Internet-facing ALBs and NAT gateways |
| Private subnets | Three `/20` subnets across available AZs | EKS managed nodes and workloads |
| Intra subnets | Three `/20` subnets across available AZs | EKS control-plane network interfaces |
| Internet gateway | VPC module-managed | Public-subnet internet route |
| NAT gateways | One in portfolio mode; one per AZ in production mode | Outbound-only private workload access |
| Route tables | Separate public, private, and intra tiers | Prevent accidental control-plane/public routing overlap |
| Security groups | Separate EKS cluster and node groups | Control plane/node traffic with module-maintained EKS rules |
| VPC Flow Logs | CloudWatch, 60-second aggregation, 30-day retention | Network audit and troubleshooting evidence |

Public subnets receive `kubernetes.io/role/elb`; private subnets receive
`kubernetes.io/role/internal-elb`. Terraform outputs list every subnet, gateway,
route-table, and security-group ID for audit evidence.

## 2. IAM and access

No long-lived AWS access key is placed in a pod or Kubernetes Secret.

| Principal | Access mechanism | Scope |
| --- | --- | --- |
| Human administrator | EKS access entry | Optional IAM Identity Center/dedicated role; cluster-admin policy |
| EKS control plane | Module-created IAM role | Required EKS service actions |
| Managed nodes | Module-created instance role | Node bootstrap and ECR/network baseline |
| VPC CNI | EKS Pod Identity | AWS VPC CNI policy, associated to `kube-system/aws-node` |
| EBS CSI | EKS Pod Identity | EBS CSI policy, associated to `kube-system/ebs-csi-controller-sa` |
| Load Balancer Controller | EKS Pod Identity | ELB/EC2 controller policy, associated to its service account |
| External Secrets | EKS Pod Identity | Read-only Secrets Manager access under `kubevista/` by default |

`enable_cluster_creator_admin_permissions` is disabled. Human access is
declarative, and IAM users are rejected by variable validation. Pod Identity
roles trust `pods.eks.amazonaws.com`; the Pod Identity agent delivers temporary
credentials through the AWS SDK default credential chain.

## 3. EKS control plane

- Kubernetes `1.36`, with the minor version explicit in Terraform.
- Private API endpoint enabled. Public API access is disabled unless explicit
  CIDRs are supplied; the example rejects the common `0.0.0.0/0` practice.
- Authentication mode is the EKS API, not the legacy `aws-auth` ConfigMap.
- API, audit, authenticator, controller-manager, and scheduler logs are retained
  in CloudWatch for 30 days.
- Kubernetes Secrets use KMS envelope encryption.
- Control-plane ENIs use isolated intra subnets; nodes use private subnets.

## 4. Compute

The baseline uses an EKS managed node group with two on-demand AL2023 x86 nodes,
scaling from two to four across AZs. `t3.large` and `t3a.large` are allowed so
the group can choose between equivalent capacity pools.

Nodes require IMDSv2 and set the metadata hop limit to one, preventing ordinary
pods from inheriting the node role. Rolling updates permit at most 33 percent
unavailable. Workloads use topology spread constraints, disruption budgets,
requests/limits, health probes, non-root users, read-only filesystems, and
dropped Linux capabilities.

Managed nodes are intentionally the first compute profile. Karpenter is useful
after workload shape, interruption tolerance, and scaling SLOs are measured; it
is not added merely to increase the tool count.

## 5. Core Kubernetes add-ons and CNI

EKS manages lifecycle and compatibility for:

- `vpc-cni` before compute, with its own Pod Identity role;
- `kube-proxy` for Service networking;
- `coredns` for cluster DNS;
- `eks-pod-identity-agent` before compute;
- `aws-ebs-csi-driver` with its own Pod Identity role.

AWS VPC CNI is the baseline CNI because pods receive VPC-routable addresses and
AWS supports its lifecycle on EKS. Kubernetes NetworkPolicy restricts KubeVista.
Cilium/Hubble is a documented future profile: it should be introduced only with
an explicit choice between VPC CNI chaining/policy mode and full CNI replacement,
plus connectivity, upgrade, and rollback tests.

The `gp3` StorageClass uses encrypted EBS volumes, waits for pod scheduling
before provisioning in an AZ, permits expansion, and retains volumes after PVC
deletion.

## 6. AWS integrations and GitOps

Argo CD `10.4.2` is bootstrapped once. Its root application creates an AppProject
and child applications. Automated sync has pruning and self-healing enabled;
server-side apply handles large CRDs. Exact chart versions are pinned:

| Integration | Chart version | Namespace |
| --- | ---: | --- |
| AWS Load Balancer Controller | `3.5.0` | `kube-system` |
| External Secrets Operator | `2.10.0` | `external-secrets` |
| Metrics Server | `3.14.0` | `kube-system` |
| kube-prometheus-stack | `88.6.1` | `observability` |
| Loki | `18.11.7` | `observability` |
| Tempo | `2.3.0` | `observability` |
| OpenTelemetry Collector | `0.172.0` | `observability` |

The Load Balancer Controller finds the VPC by its Terraform tag, so it does not
need pod access to node metadata. External Secrets uses controller Pod Identity;
its ClusterSecretStore deliberately has no `auth` block. The example
`platform/examples/external-secret.yaml` reads `kubevista/example` only after an
operator creates that AWS secret.

KubeVista's ALB Ingress is disabled by default so repository validation cannot
create a billable load balancer. Enabling it creates an internal ALB with IP
targets and a `/healthz` check; the NetworkPolicy then permits only the VPC CIDR
to the API port. Public exposure requires a separate authentication and threat
model decision.

## 7. Applications and testing

KubeVista is a Go `client-go` application, not a static mock. In cluster mode it
uses its service-account token and read-only ClusterRole to count nodes,
namespaces, and pod phases. Readiness verifies Kubernetes API access with a
three-second deadline. Local demo mode uses deterministic sample data.

Testing layers:

- fake-client Kubernetes inventory unit tests;
- HTTP contract tests for health, summary, and Prometheus metrics;
- `go vet` and a static Linux build;
- React/TypeScript production compilation;
- Helm lint and render for every local and third-party chart;
- Terraform formatting, dependency initialization, and validation;
- a hardened Helm smoke-test pod that calls `/healthz` through the Service;
- GitHub Actions repeats all repository-only checks on every pull request.

## 8. Monitoring and observability

```text
Go API --OTLP traces--> OTel gateway --> Tempo
Pods ---container logs-> OTel agent ----> Loki
Nodes --host/kubelet----> OTel agent ----> OTel gateway
Cluster--state/events---> OTel gateway --> Prometheus exporter
Go API --/metrics-----------------------> Prometheus
Prometheus + Loki + Tempo --------------> Grafana data sources
```

The Go API uses W3C trace context and baggage propagation, batches OTLP/gRPC
exports, and samples 25 percent by default. When no OTLP endpoint is configured,
telemetry is a no-op so local development remains dependency-free. `/metrics`
exposes Go runtime and process metrics for the ServiceMonitor.

The portfolio profile keeps seven days/20 GiB of Prometheus data, seven days/20
GiB of Loki data, and 24 hours/10 GiB of Tempo data. Grafana uses a 5 GiB volume,
and Alertmanager uses 5 GiB. Loki and Tempo are single replicas. A production
profile must move logs/traces to versioned object storage, add multi-AZ replicas,
configure Alertmanager receivers, use SSO for Grafana, and test restore and
retention enforcement.

## Security and cost boundaries

- CI never runs `terraform apply` and does not need AWS credentials.
- Terraform state uses a versioned, KMS-encrypted, TLS-only S3 bucket with native
  lock files and `prevent_destroy`.
- Private endpoint access needs VPN/VPC connectivity; temporary public access is
  restricted to an operator `/32`.
- The single NAT gateway lowers demo cost but is an AZ dependency. Set
  `single_nat_gateway = false` for the production profile.
- EKS, NAT gateways, nodes, load balancers, EBS volumes, and CloudWatch ingestion
  all cost money. The Terraform budget is an alert, not a hard spending cap.
- Destroy the EKS environment after demonstrations; preserve remote state.

## Source-of-truth ownership

Terraform owns AWS network, IAM, EKS, compute, EKS add-ons, and the budget. Argo
CD owns in-cluster controllers and workloads. Helm owns each application's
Kubernetes objects. Mixing those boundaries—for example installing EBS CSI both
with Terraform and Helm—would create conflicting lifecycles and is prohibited.
