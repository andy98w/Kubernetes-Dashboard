# Public delivery and authentication

During the August 31 deployment, `kubevista.illuma.me` pointed to an
internet-facing AWS Application Load Balancer. The root domain stayed on
Vercel, while the KubeVista subdomain used its own Route 53 hosted zone. The
ALB and delegation records were removed during teardown, so that endpoint is
offline now.

## Request path

1. Vercel delegated the subdomain with four NS records.
2. Route 53 was authoritative for the subdomain and held the ACM validation
   CNAME and an Amazon-authorizing CAA record.
3. ExternalDNS watched the KubeVista Ingress and created the ALB alias plus its
   TXT ownership record. Its IAM policy can change records only in the
   delegated hosted zone, and credentials arrive through EKS Pod Identity.
4. The ALB redirected HTTP to HTTPS, used a TLS 1.2/1.3 policy, and authenticated
   every request through Amazon Cognito before forwarding to the web pods.
5. The web NetworkPolicy accepted ALB traffic only from the three public-subnet
   CIDRs. An in-cluster negative test confirmed that an untrusted pod could not
   reach the application.

## Identity controls

The Cognito user pool allowed only administrator-created users, used email
usernames, required a 14-character mixed password, and required software-token
MFA. The ALB session lasted one hour. The Load Balancer Controller could call only
`cognito-idp:DescribeUserPoolClient` against this specific pool.

The administrator email came from an ignored Terraform variable. No password,
client secret, session cookie, or access token is stored in Git.

## Certificate controls

The parent domain had CAA records for other certificate authorities, so the
delegated Route 53 zone published `0 issue "amazon.com"` at its apex. The ACM
validation record existed during the deployment and was removed with the zone.

## Cost and lifecycle

ACM public certificates are free when used with integrated AWS services.
Route 53 hosted zones and ALB runtime incur charges. The ALB carried the
environment name and teardown deadline as tags. The runbook deletes the Ingress
first and waits for the controller to remove the ALB before Terraform destroys
the VPC.
