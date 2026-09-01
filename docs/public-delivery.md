# Public delivery and authentication

KubeVista is published at `https://kubevista.illuma.me` through an
internet-facing AWS Application Load Balancer. The root domain remains on
Vercel; only the `kubevista.illuma.me` subdomain is delegated to a dedicated
Route53 public hosted zone.

## Request path

1. Vercel delegates the subdomain with four NS records.
2. Route53 is authoritative for the subdomain and retains the ACM validation
   CNAME and an Amazon-authorizing CAA record.
3. ExternalDNS watches the KubeVista Ingress and creates the ALB alias plus its
   TXT ownership record. Its IAM policy can change records only in the
   delegated hosted zone, and credentials arrive through EKS Pod Identity.
4. The ALB redirects HTTP to HTTPS, uses a TLS 1.2/1.3 policy, and authenticates
   every request through Amazon Cognito before forwarding to the web pods.
5. The web NetworkPolicy accepts ALB traffic only from the three public-subnet
   CIDRs. Untrusted pods in the private node subnets remain blocked.

## Identity controls

The Cognito user pool is administrator-created-user-only, uses email usernames,
requires a 14-character mixed password, and requires software-token MFA. The
ALB session lasts one hour. The Load Balancer Controller can call only
`cognito-idp:DescribeUserPoolClient` against this specific pool.

The initial administrator email is an ignored Terraform variable. No password,
client secret, session cookie, or access token is stored in Git.

## Certificate controls

The parent domain has restrictive CAA records for other certificate
authorities. The delegated Route53 zone therefore publishes
`0 issue "amazon.com"` at its apex. The DNS validation CNAME must remain after
issuance so ACM can renew the certificate automatically.

## Cost and lifecycle

ACM public certificates are free when used with integrated AWS services.
Route53 hosted zones and ALB runtime incur charges. The ALB is tagged with the
portfolio environment and teardown deadline. Delete the Ingress and wait for
the controller to remove the ALB before destroying the VPC; retain the hosted
zone only if the public hostname should survive between demo runs.
