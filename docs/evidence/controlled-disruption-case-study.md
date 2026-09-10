# Controlled disruption case study

## Scenario

KubeVista's API and web tiers each ran with two replicas, a PodDisruptionBudget,
rolling-update safeguards, health probes, and topology spreading. To validate
that those controls worked together, one running pod from each tier was removed
while Fortio continuously sent traffic through the Kubernetes Service.

## Hypothesis

The surviving replica should continue serving requests while the Deployment
controller creates a replacement. Readiness should prevent traffic from
reaching the replacement until it can serve successfully.

## Observed signals

| Signal | API test | Web test |
| --- | --- | --- |
| Traffic duration and rate | 45 seconds at 20 requests/second | 45 seconds at 20 requests/second |
| Replacement readiness | 2 seconds | 2 seconds |
| Successful responses | `896/896` HTTP 200 | `896/896` HTTP 200 |
| Average latency | `0.741 ms` | `2.597 ms` |
| Approximate p99 | `2.78 ms` | `8.51 ms` |

## Impact and recovery

No request failure was observed during either controlled single-pod loss. The
Deployment controller restored the desired replica count, and readiness gating
kept the replacement out of service until it was healthy.

## Why this is useful evidence

This test validates a complete behavior rather than the presence of individual
YAML fields: replica count, Service routing, probes, scheduling, and controller
reconciliation all contributed to continuity. It does not prove tolerance of a
node or Availability Zone outage, control-plane failure, correlated dependency
failure, or sustained peak traffic.

## Follow-up production tests

1. Drain a node and verify PDB behavior and cross-node rescheduling.
2. Exercise an Availability Zone failure with sufficient capacity in the
   remaining zones.
3. Run a longer soak test while collecting CPU, memory, saturation, and error
   budget data from Prometheus.
4. Inject downstream Kubernetes API latency and verify KubeVista's timeout,
   stale-data, and stream-reconnect behavior.

The raw summarized measurements are retained in
[the live deployment record](live-eks-2026-08-31.md#load-and-recovery-evidence).
