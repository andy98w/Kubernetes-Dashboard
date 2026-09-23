# KubeVista delivery

## Implemented path

1. PR CI runs Go race tests/vet, durable-operation integration tests, frontend
   checks and browser tests, container builds, Helm validation and Terraform validation.
2. A main application change runs that same CI before publishing both images.
   The publisher uses the existing `kubevista-images` environment and AWS OIDC
   role. ECR images carry a commit tag, BuildKit SBOM and provenance. Trivy scans
   the pushed digest and fails on HIGH/CRITICAL vulnerabilities, including unfixed
   findings. Only passing images are signed by Cosign and included in artifacts.
   Failed scans can leave an unsigned image in ECR, but cannot promote it.
3. Both artifacts must have the same commit and valid ECR digest references.
   A new PR updates `platform/releases/staging.json`. The workflow dispatches CI
   explicitly because token-created PRs don't trigger PR workflows on their own.
4. Merging the staging PR lets the opt-in staging Argo Application reconcile.
   A PostSync Job checks API readiness and the web page. It uses the smoke-test
   network-policy allowance, not the web label (which would make it a backend).
5. Manually dispatch production promotion on main after verifying staging.
   The `kubevista-production` environment must require a reviewer. The job copies
   the staging digests unchanged into a production PR. Production Argo sync is
   manual as a second gate. There is no automatic rollback: revert the release
   commit and sync the previous desired state.

Tag/manual publishing also produces artifacts, but only main creates staging PRs.
Duplicate immutable commit tags may reject a rerun; do not disable immutability
to work around that. Inspect the existing run/artifacts or publish a new commit.

## Activation checklist — not performed locally

- Enable Actions PR creation, and protect main with required CI checks and review.
  Protect `.github/workflows`, delivery scripts and release values with review.
- Set `AWS_IMAGE_PUBLISHER_ROLE_ARN`, `AWS_REGION`, `AWS_ACCOUNT_ID` on the
  existing images environment. Its IAM trust must match the exact environment
  OIDC subject. Keep release permissions separate from infrastructure deployment.
- Create `kubevista-production` with required reviewers and main-only deployment
  branches. A YAML environment name alone does **not** enforce approval.
- Provision/register distinct staging and production clusters with Argo, ECR
  pull access, the observability CRDs/collector, and environment-specific network
  settings. Review the chart's API service CIDR; historical defaults are not
  portable. Ingress is deliberately disabled in the example applications.
- Merge an actual publisher-generated staging release before installing
  `platform/examples/delivery-applications.yaml`. That file is outside the watched
  app directory; nothing is silently installed. The chart uses fixed ClusterRole
  names, so these two releases must not share one cluster. Do not install alongside
  the existing dashboard Application on a destination cluster.
- Confirm PostSync success and readiness before approving production. The workflow
  does not independently attest staging health. Keep that as a human approval
  criterion until an authenticated deployment-evidence collector is configured.

No AWS resources were created, settings changed, release published or PR pushed
when implementing this feature. These workflows need an actual GitHub run to
validate OIDC, ECR, scanning and promotion end to end.

## Station representation

The external GitHub dock, CI workshop and ECR warehouse lead to an Argo dispatch
desk. Purple instructions and gold image-pull lines have separate meanings.
They are schematic, not measured traffic. Click the buildings to inspect delivery.
Demo mode offers a successful release and a failed readiness rehearsal; the
version crate moves through stages and never starts an external build.

Live mode polls `/api/v1/delivery` every 15 seconds. Kubernetes observations include
image references and Deployment updated/ready counts. Old observed generations
cannot count as complete; progress deadline failures remain failures. These are
Deployment status observations, not proof that CI or signatures succeeded.

Optional chart flags:

- `deliveryGitHub: true` enables a read-only public GitHub API request for the
  latest main `images.yml` run. No token is sent to the browser. Rate limiting or
  network failure leaves status unknown. The current default egress policy blocks
  external HTTPS; explicitly provide a reviewed GitHub egress path to enable it.
- `deliveryArgo: true` grants GET on the single `argocd/kubevista-dashboard`
  Application and reads sync/health status. It is for the existing in-cluster
  Application, not a remote central Argo instance. The staged multi-cluster
  example needs an authenticated central observer before Argo status can be read.

Publisher status, Argo revision and running images are not automatically correlated
into one release receipt. ECR signature verification in admission, canary analysis
with Argo Rollouts, migration orchestration and automatic rollback are not enabled.
Those are follow-on capabilities, not claims made by this map.

## Local checks

```sh
python3 -m unittest discover -s scripts/delivery -p 'test_*.py'
helm lint platform/apps/dashboard --set deliverySmoke=true
cd api && go test ./internal/kubernetes ./internal/httpapi
```

Reference behavior: [GitHub reusable workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows),
[Trivy image scanning](https://trivy.dev/docs/dev/references/configuration/cli/trivy_image/).
