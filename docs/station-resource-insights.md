# Resource relationships in the station

Click a robot to highlight pods belonging to the same namespace, controller kind,
and workload name. Selection moves each room to a page containing a sibling.
The inspector keeps the workload-detail link. Click the robot again, choose All,
or press Escape on the map to clear the selection.

Service lines come from EndpointSlice membership. Known fleet dependencies
highlight alongside the selected workload; these are configured relationships,
not evidence of observed requests. Unknown outbound callers are not inferred.

Each node has a small CPU/memory meter. Live usage comes from Metrics Server;
samples older than two minutes are ignored. CPU is compared in millicores and
memory in bytes against node allocatable capacity. Missing samples appear as
question marks, not zero. The indicator reports MemoryPressure, DiskPressure,
and PIDPressure conditions; it is not an overall node-health verdict.

Storage cabinets appear for mounted PVC and emptyDir volumes. Inspection lists
pod, namespace, claim, capacity, storage class, and claim phase. A PVC's lifetime
is independent of a pod; emptyDir lasts for that pod. This inventory does not
claim to cover every CSI, hostPath, or projected volume type. An unavailable PVC
lookup leaves claim metadata unknown. Other storage types are not rendered.

Unscheduled, nonterminal pods use a separate waiting platform. Inspection shows
the PodScheduled condition's reason and message. They are not counted as worker
nodes or assigned a cloud subnet. Up to nine waiting robots are rendered, with
an overflow count and the full workload inventory available.

AZ outlines use only the node's topology.kubernetes.io/zone label. Matching
colors identify matching zones. Unlabeled nodes receive no AZ outline.

The read-only chart roles now include persistentvolumeclaims and
metrics.k8s.io/nodes. A chart rollout is required for an existing deployment.
No infrastructure is provisioned by these UI changes.

The offline fleet has no invented node usage, AZ labels, or PVC mounts. These
data-dependent objects appear when a connected API supplies the corresponding
facts. Backend tests cover namespace-scoped claims, mounted-only volumes,
scheduling reasons, pressure, capacity units, and missing metrics.
