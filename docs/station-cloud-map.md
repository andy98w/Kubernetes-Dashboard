# Fleet map infrastructure layer

The fleet view (`?fleet=1`) is an architecture workspace. It does not provision
AWS resources or discover an AWS account. The connected cluster view does not
render the fleet's planned cloud boundaries.

## Service routing

The Go topology endpoint now returns the full Service list, including headless
Services and Services with no EndpointSlice entries. The map draws one desk per
Service, keyed by namespace/name. Nine desks fit on each page; subsequent pages
remain accessible from the room. Selecting a desk highlights matching loaded
EndpointSlice pod references, not merely matching names across namespaces.

Ingress host/path rules lead from the shared ALB to their individual Service
desks. Internal Services do not gain an Internet route just because they exist.
The drawing expresses Kubernetes routing relationships, not a physical Service
proxy hop or observed packet traffic. Paths are animated illustrations.

## Cloud scopes

The outlined AWS region uses the Terraform default, us-west-2. Private worker
subnet outlines group workers without guessing their AZ/subnet assignment.
Public and private outlines refer to the same VPC. Terraform defines three AZs,
public/private/intra subnet classes, and a single NAT by default. Managed EKS
control-plane hosts are outside the customer VPC; their drawn suite is conceptual.
ECR is an AWS managed registry in the separate delivery lane, not in a subnet.

Incoming: Internet → Internet gateway → shared ALB → configured Ingress backend
Service → EndpointSlice pod targets. The map's single ALB is a fleet design choice,
not proof that every app has an applied shared IngressGroup.

Outbound: private workload → NAT gateway → Internet gateway → external service.
The NAT does not receive inbound app requests. Single-NAT operation trades AZ
resilience for cost; the Terraform variable permits per-AZ NAT instead.

## Illuma storage

The object-store cabinet represents a requested fleet dependency. A scan of the
local Illuma checkout did not establish its storage provider, bucket, or upload
flow. It remains outside the AWS outline and is not labeled S3. The API-to-storage
path is a design placeholder: confirm whether uploads are proxied by the API or
use browser-side presigned URLs before treating this path as implemented. No S3
gateway endpoint is asserted.

## Verification

Unit coverage checks namespace isolation, empty/headless Service retention, desk
placement, and Go Service serialization. Browser checks cover selecting an app
Service and inspecting ALB/NAT details. No production deployment is part of this
change.
