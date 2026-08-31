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

No AWS access key is stored in GitHub. The publisher role can request an ECR
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
