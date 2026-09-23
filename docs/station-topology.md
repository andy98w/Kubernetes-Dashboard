# Station topology discovery

`GET /api/v1/topology` is a read-only inventory snapshot. It returns:

- Ingress default and HTTP backends, including host, path, Service and port.
- EndpointSlice addresses and Pod target references, preserving unknown and false readiness.
- Pod service accounts and NetworkPolicies whose selectors match their labels.
- Declared RBAC grants for service-account, user and standard service-account group subjects.
- ExternalName Service destinations.

No Secrets, service-account tokens or container environments are read.
Read failures produce section warnings, not an assertion that no resources or grants exist.
RBAC grants are configuration evidence, not SubjectAccessReview results. They do not
include cloud IAM permissions or prove effective authorization. NetworkPolicy specs
are not evaluated for connectivity or CNI enforcement.

The station refreshes live discovery every 30 seconds and clears its snapshot on
fetch failure. The demo uses explicit synthetic fixtures and never falls back to
them after a live error. Click the Internet booth and select a route to connect
the Service desk to visible pod robots referenced by EndpointSlices. Unready
endpoints remain included; lines mean membership, not successful requests.

Internet, Policy and Egress objects sit above the station. Selecting an Ingress
route connects its booth to the routing desk and visible backend robots. Green
solid endpoint links mean ready; amber dashed links mean unready; gray dashed
links mean unknown or conflicting readiness across slices. These indicators
describe the discovery snapshot, not request success. Routes disappear when the
snapshot is unavailable or the selected backend is no longer in its routes.

Connections are enabled initially. All discovered Ingress backends appear until
a route is selected; “All routes” restores the full set. The Connections switch
hides both internal and boundary links. Peach dotted Policy links indicate a
policy selector matching a visible pod, never an allow/deny verdict. Mint
dash-dot Egress links connect the Service desk to the exit only when ExternalName
destinations exist; they do not identify callers or prove outbound traffic.
Click either connection to inspect its inventory. Missing discovery data produces
no link. The current demo has no pod-policy associations or ExternalName records,
so those objects remain unconnected rather than implying unsupported relationships.

The checked-in telemetry configuration does not currently enable Tempo service
graph generation. Measured service-call pulses and latency/error indicators need
that metrics source (or another observed-call source) plus an explicit mapping
from telemetry service identities to Kubernetes workloads. No synthetic request
activity is used as a live fallback.

RBAC discovery is opt-in for the dashboard chart:
`topologyRBACDiscovery: true` adds list-only access to roles and bindings.
Without it, other discovery works and RBAC reports incomplete inventory.
No cluster changes are applied by editing the chart.

Still outstanding: Gateway API discovery, effective authorization checks,
connection-specific NetworkPolicy evaluation, measured service-to-service traffic,
and latency/error/request pulses. ExternalName records do not identify callers.
The existing telemetry room depicts configured pipelines, not measured flows.
