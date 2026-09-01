# Platform runbook

## Validate without AWS

```bash
make test
make terraform-check
make platform-check
```

`platform-check` downloads only the exact chart versions referenced by the Argo
CD applications and renders them with the committed values.

## Deploy

1. Use an AWS IAM Identity Center session and verify the intended account:
   `aws sts get-caller-identity`.
2. Apply the remote-state bootstrap described in `docs/deployment.md`.
3. Review and apply the saved `dev.tfplan`.
4. Run the `configure_kubectl` Terraform output from a network path that can
   reach the EKS API endpoint.
5. Install Argo CD `10.4.2` with `platform/argocd/values.yaml`.
6. Apply `platform/argocd/root-application.yaml`.

Publish application images before the dashboard sync. The release workflow
prints both signed digests; set the Helm image repositories/tags to those
artifacts or pin the full digest in an environment override before promotion.

Do not apply the example ExternalSecret until the source secret exists:

```bash
aws secretsmanager create-secret \
  --name kubevista/example \
  --secret-string '{"api-key":"replace-me"}'
kubectl apply -f platform/examples/external-secret.yaml
```

Never put a real secret value in Git or Terraform variables.

## Ephemeral portfolio lifecycle

The production-shaped environment is intentionally run on demand. Use an
eight-hour default TTL, preserve the encrypted state backend, and destroy the
chargeable environment after each demonstration.

For a local SSO session:

```bash
export AWS_PROFILE=kubevista
export KUBEVISTA_AWS_ACCOUNT_ID=<expected-account-id>
export KUBEVISTA_TTL_HOURS=8
scripts/terraform-ephemeral.sh plan
scripts/terraform-ephemeral.sh apply
```

The plan adds an `ExpiresAt` tag to tagged AWS resources and writes the deadline
to ignored local `work/ephemeral-deployment.env`. A tag is evidence and an
operator signal; it does not automatically delete resources.

The manual `Ephemeral EKS lifecycle` GitHub workflow offers the same
plan/apply/destroy controls. Its `kubevista-ephemeral` GitHub Environment must
require approval and define these environment values:

- variables: `AWS_ACCOUNT_ID`, `AWS_REGION`, `AWS_DEPLOY_ROLE_ARN`,
  `EKS_ADMIN_PRINCIPAL_ARN`, `EKS_PUBLIC_ACCESS_CIDRS_JSON`, `TF_STATE_BUCKET`,
  and `TF_STATE_KMS_KEY_ARN`;
- secret: `BUDGET_NOTIFICATION_EMAIL`.

`AWS_DEPLOY_ROLE_ARN` must trust GitHub's OIDC provider only for this repository
and environment. Do not store AWS access keys in GitHub. A workflow invocation
must also contain the exact typed confirmation `PLAN kubevista-dev`,
`APPLY kubevista-dev`, or `DESTROY kubevista-dev`. Concurrency prevents two
Terraform lifecycle operations from mutating the same state simultaneously.

## Verify each layer

```bash
terraform -chdir=infra/terraform/environments/dev output
aws eks describe-cluster --name kubevista-dev --query 'cluster.status'
kubectl get nodes -o wide
kubectl get pods -n kube-system
aws eks list-pod-identity-associations --cluster-name kubevista-dev
kubectl get applications -n argocd
kubectl get storageclass gp3
kubectl get pods,pvc -n observability
kubectl get servicemonitors -A
helm test kubevista -n kubevista
kubectl get deploy,svc,hpa,pdb,networkpolicy -n kubevista
```

Expected results: the cluster is `ACTIVE`; two nodes are `Ready`; all Argo CD
applications are `Synced` and `Healthy`; gp3 is the default StorageClass; and
the KubeVista Helm test succeeds.

## Verify public delivery

```bash
dig NS kubevista.illuma.me
aws acm describe-certificate --certificate-arn <arn> \
  --query 'Certificate.Status'
kubectl get ingress -n kubevista kubevista-web
kubectl logs -n external-dns deployment/external-dns --tail=100
curl -I http://kubevista.illuma.me
curl -Ik https://kubevista.illuma.me
```

Expected results: delegation returns the Route53 name servers, ACM is `ISSUED`,
the Ingress has an ALB hostname, ExternalDNS reports an upsert for the A/AAAA
alias and TXT ownership record, HTTP redirects to HTTPS, and an unauthenticated
HTTPS request redirects to the Cognito hosted UI.

The first administrator signs in with the temporary Cognito credential sent by
email, chooses a permanent password, and enrolls a software TOTP authenticator.
Never record that credential or TOTP seed in Git, screenshots, or logs.

## Access dashboards safely

Keep admin interfaces private. Use port forwarding for a demo:

```bash
kubectl port-forward -n argocd service/argocd-server 8081:443
kubectl port-forward -n observability service/monitoring-grafana 3000:80
```

Retrieve generated bootstrap passwords from their Kubernetes Secrets only at
the terminal; do not paste them into tickets, chat, screenshots, or commits.

## Common failures

| Symptom | Checks |
| --- | --- |
| Terraform cannot reach EKS | Private endpoint requires VPN, bastion, or VPC runner; temporarily allow only your `/32` if approved |
| Pods remain Pending with PVCs | Check EBS CSI add-on, its Pod Identity association, node AZ capacity, and `kubectl describe pvc` |
| LoadBalancer/Ingress is not reconciled | Check controller service-account name, Pod Identity association, VPC tags, subnet tags, and controller logs |
| ExternalSecret reports authentication failure | The ClusterSecretStore must have no `auth` block for Pod Identity; verify association and secret prefix |
| Grafana has no OTel data | Verify OTel collector health, service DNS, network policy, ServiceMonitor discovery, and exporter errors |
| KubeVista readiness is 503 | The service account cannot list namespaces or the API is unavailable; inspect RBAC and API audit logs |
| Web health is 502/503 | Check API endpoints, web-to-API NetworkPolicy, CoreDNS, and NGINX logs |
| Argo application is OutOfSync | Inspect diff before syncing; confirm chart version and CRD ownership; do not force-delete production CRDs |

## Teardown

Treat teardown as an ordered controller and ownership workflow, not just a
Terraform command. Kubernetes controllers can recreate resources while they
are running, and dynamically provisioned resources may not exist in Terraform
state.

First stop Argo CD reconciliation, delete the public Ingress, and wait until the
AWS Load Balancer Controller has removed the ALB and target groups:

```bash
kubectl scale statefulset argocd-application-controller \
  --namespace argocd --replicas=0
kubectl delete ingress kubevista-web --namespace kubevista
aws elbv2 describe-load-balancers \
  --query 'LoadBalancers[?contains(LoadBalancerName, `kubevista`)].LoadBalancerArn'
```

Do not proceed while the final command returns a KubeVista load balancer. If
the cluster is already unavailable, inspect and remove controller-owned load
balancer resources explicitly before attempting VPC deletion.

Then create, inspect, and apply a saved destroy plan:

```bash
terraform -chdir=infra/terraform/environments/dev plan -destroy -out=destroy.tfplan
terraform -chdir=infra/terraform/environments/dev apply destroy.tfplan
```

The guarded local equivalent is:

```bash
AWS_PROFILE=kubevista \
KUBEVISTA_AWS_ACCOUNT_ID=<expected-account-id> \
scripts/terraform-ephemeral.sh destroy
```

The state bucket is intentionally retained. Confirm EBS volumes, load balancers,
NAT gateways, and CloudWatch log groups according to the chosen retention policy
so no unexpected recurring charges remain.

Run the repository verifier after Terraform completes:

```bash
AWS_PROFILE=kubevista AWS_REGION=us-west-2 \
  scripts/verify-aws-cleanup.sh
terraform -chdir=infra/terraform/environments/dev state list
```

The second command must print no resources. The verifier uses live EC2 and ELB
APIs; do not treat results from the eventually consistent Resource Groups
Tagging API alone as proof that a resource still exists.

### Controller-owned Route 53 records

ExternalDNS-created A, AAAA, and TXT records are not Terraform resources. If
Route 53 returns `HostedZoneNotEmpty`, list the record sets, distinguish the
required NS/SOA records from controller-owned application records, and delete
only the inspected application records. Rerun a saved destroy plan and verify
that it contains only the hosted zone before applying it. Do not enable broad
zone deletion merely to suppress the ownership check.

After the hosted zone is deleted, remove the corresponding subdomain NS
delegation from the parent DNS provider. A newly created Route 53 zone will have
a new nameserver set, so keeping or restoring the old values creates a dangling
delegation.

### Dynamically provisioned EBS volumes

PVC volumes are created by EBS CSI rather than Terraform and can survive the
cluster depending on reclaim behavior and teardown timing. Inspect them through
the authoritative EC2 API:

```bash
aws ec2 describe-volumes \
  --filters Name=tag:Project,Values=KubeVista \
  --query 'Volumes[].{id:VolumeId,state:State,size:Size,attachments:Attachments,tags:Tags}'
```

Delete a volume only after verifying it is `available`, unattached, and tagged
for the destroyed cluster and expected PVC. Volume deletion is irreversible;
retain or snapshot data when the environment's retention policy requires it.

The first teardown, including the ExternalDNS ownership issue, orphaned
observability volumes, SSO recovery, parent-zone cleanup, and zero-resource
verification, is recorded in
[`docs/evidence/teardown-2026-08-31.md`](evidence/teardown-2026-08-31.md).

Keep static portfolio sites outside EKS (for example, on static object/CDN
hosting). The cluster is reserved for demonstrations that benefit from
Kubernetes scheduling, IAM, GitOps, and observability; placing a static site in
EKS does not reduce the fixed EKS or NAT gateway charges.
