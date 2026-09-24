# Portfolio cluster recreation review — September 23, 2026

Status: planned, not applied. No website deployment or DNS change has occurred.

The authenticated AWS check found no `kubevista-dev` cluster in `us-west-2`.
The existing remote dev state contains no managed resources. Terraform produced
`infra/terraform/environments/dev/website-review.tfplan` with **91 additions,
zero changes, and zero deletions**. The plan and operator overrides are ignored
by Git; treat the binary plan as sensitive.

## Reviewed scope

- EKS 1.36, confirmed in standard support by the regional EKS API.
- Three-AZ VPC: three private, three public, and three control-plane subnets.
- Two on-demand workers, instance options t3.large and t3a.large; max four.
- One NAT gateway and one public IPv4 address. This is a cost-saving,
  single-AZ egress dependency, not fully AZ-resilient outbound connectivity.
- Five managed add-ons, IAM/Pod Identity, security groups, KMS and 30-day logs.
- Public Kubernetes API restricted to the operator's current /32; private access enabled.
- No Route53 zone, Cognito, certificate, public load balancer, or website DNS changes.

The override disables the retired public-delivery configuration, replaces the
old allowed IP, and removes the expired teardown tag. Choose a new teardown
deadline and regenerate the plan before applying. An expiry tag does not itself
delete resources. The existing $100 monthly budget sends alerts, not a spending cap.

## Cost estimate

USD list prices checked September 23, 2026; 730 hours/month, two workers, no
credits or discounts. EC2, gp3 and Oregon NAT rates came from AWS Pricing API.

| Item | Monthly estimate |
| --- | ---: |
| Standard-support EKS, $0.10/hour | $73.00 |
| Two t3a.large–t3.large workers, $0.0752–$0.0832 each/hour | $109.79–$121.47 |
| One NAT gateway, $0.045/hour | $32.85 |
| One public IPv4 address, $0.005/hour | $3.65 |
| Two 20 GiB gp3 root disks, $0.08/GiB-month | $3.20 |
| One new KMS key | $1.00 |
| **Fixed baseline** | **$223.49–$235.17** |

The disk estimate uses the currently recommended EKS AL2023 image's 20 GiB gp3
root volume; the launch template does not explicitly pin disk sizing. Approximate
baseline is $0.31–$0.32/hour or $7.35–$7.73/day, before usage charges.

Excluded: CloudWatch ingestion/storage, NAT processing ($0.045/GB), cross-AZ and
internet transfer, ECR/image builds, existing state storage/KMS, KMS requests,
additional persistent volumes, CPU surplus credits, taxes and extra nodes.
Grafana/Prometheus/Loki/Tempo and Argo CD are not installed by this Terraform
plan. Installing them can add storage, traffic, and compute needs. No public
ALB is included. The portfolio application image still needs to be built and tested.

Sources: [EKS](https://aws.amazon.com/eks/pricing/),
[EC2](https://aws.amazon.com/ec2/pricing/on-demand/),
[VPC](https://aws.amazon.com/vpc/pricing/),
[EBS](https://aws.amazon.com/ebs/pricing/),
[KMS](https://aws.amazon.com/kms/pricing/).

## Recommendation

Pre-apply security finding: EBS encryption-by-default is disabled in this region,
the recommended AMI root snapshot is unencrypted, and the launch template has no
explicit encrypted disk mapping. Add an encrypted root-volume mapping and replan
before approval. The cluster's KMS secret encryption does not encrypt node disks.

Use a short-lived cluster for the deployment demonstration and keep the live
portfolio on its existing hosting. Do not run this environment continuously just
for the static website. Approve a duration and expected spend before recreation;
regenerate/review the plan with a fresh deadline and operator IP, then explicitly
apply. Teardown must include any later Kubernetes-created AWS resources.
