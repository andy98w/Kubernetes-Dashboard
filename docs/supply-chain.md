# Container supply chain

The persistent bootstrap stack owns two immutable, scan-on-push ECR
repositories: `kubevista-api` and `kubevista-web`. The ephemeral EKS stack can be
destroyed without losing release artifacts or Terraform state.

## Trust path

```text
Git tag/manual dispatch
  -> protected kubevista-images environment
  -> GitHub OIDC token (repository + environment + STS audience)
  -> narrow AWS publisher role
  -> BuildKit image + SBOM + provenance
  -> immutable ECR sha-<commit> tag
  -> ECR basic scan
  -> keyless Cosign signature on the digest
  -> reviewed digest promoted through Helm/Argo CD
```

No AWS access key is stored in GitHub. AWS trust matches GitHub's durable owner
and repository IDs as well as the protected environment, rather than relying
only on mutable display names. The publisher role can request an ECR
authorization token and upload layers only to the two application repositories;
it cannot administer ECR, read Terraform state, or change EKS.

## Operator setup

Create the protected `kubevista-images` GitHub Environment and define:

- `AWS_ACCOUNT_ID`: expected AWS account ID;
- `AWS_REGION`: bootstrap/ECR region;
- `AWS_IMAGE_PUBLISHER_ROLE_ARN`: Terraform bootstrap output.

Require reviewer approval for production publishing. Run the workflow manually
for a candidate or create a signed `v*` tag for a release. Use the digest from
the workflow summary when updating the GitOps environment.

## Verification

```bash
aws ecr describe-images \
  --repository-name kubevista-api \
  --image-ids imageTag=sha-<full-commit>

cosign verify \
  --certificate-identity-regexp '^https://github.com/andy98w/Kubernetes-Dashboard/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  <account>.dkr.ecr.<region>.amazonaws.com/kubevista-api@sha256:<digest>
```

ECR lifecycle rules delete untagged images after seven days and retain the
latest 20 `sha-`/`v` images. ECR basic scanning is a release signal, not a policy
gate yet; Kyverno digest/signature enforcement is the next control-plane step.

The dashboard chart currently promotes the `v0.2.0` build from commit `79ff0f9`
by digest. API digest
`sha256:5382dd6417b0bdcee3adbadfe6a548c81327bf028eeef2bed4e444e8290fb90c`
and web digest
`sha256:add0e4d4a0981658f5d5eb87037c4489b1c5e43c2b478aacb919eb8319927a72`
are retained in ECR. The earlier live-deployment release was independently
confirmed with active Sigstore bundle referrers; see the signed-release section
of [`evidence/live-eks-2026-08-31.md`](evidence/live-eks-2026-08-31.md).
