# Incident lab results

Measured September 11, 2026 (Pacific), on a disposable Kubernetes 1.34 kind
cluster in GitHub Actions. [Successful run and downloadable evidence](https://github.com/andy98w/Kubernetes-Dashboard/actions/runs/34672376421).

| Injected failure | Signal returned | Time to diagnosis | Diagnosis to verified rollback |
| --- | --- | ---: | ---: |
| Bad readiness path | ProbeFailed | 2 s | 2 s |
| Exiting container | CrashLoopBackOff | 5 s | 1 s |
| Missing image tag | ImagePullBackOff | 17 s | 1 s |
| Memory limit exceeded | OOMKilled | 7 s | 1 s |
| Impossible CPU request | Unschedulable | 1 s | 3 s |

These are single-run measurements with two-second diagnosis polling and integer
wall-clock timestamps, not production latency targets. Recovery includes plan,
execution, and a fresh controller-generation/replica check. The rolling-update
strategy kept the earlier healthy replica available; these numbers do not
measure recovery from a total outage or prove application-level availability.

The same run rejected reused plans, writes outside the allowed namespace, and
scaling above the replica limit. The API identity could not patch nodes. A
separate node-operator identity cordoned and uncordoned the lab node. Temporary
scaling changed one replica to two and restored one after the five-second hold;
the final check observed generation 14/14 with one updated, available replica.

The standard CI run also passed all 20 desktop/mobile browser tests, Go tests
and vet, container builds, Terraform validation, and Helm validation.

Local microbenchmark on an Apple M4 Pro: classifying 1,000 synthetic failing pods
took approximately 220 microseconds per iteration, with 378 KB allocated and
2,011 allocations. This excludes Kubernetes API calls and network latency.
The Go race-enabled package suite passed separately on the development machine.

## Limits

The new incident workflow was tested in kind, not the retired AWS cluster. The
browser failure lab is explicitly simulated. Real Prometheus request parsing is
covered with HTTP fixtures; the lab does not deploy a Prometheus server. See
[the prototype guide](incident-response.md) for install commands, data-retention
limits, and the foreground-only temporary scaling behavior.
