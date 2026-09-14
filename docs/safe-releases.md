# Safe release and automatic route rollback lab

This lab adds application-level release verification alongside KubeVista's existing
incident-response workflow. It runs only in a dedicated kind cluster. It does not
change the AWS platform, expose an unauthenticated deployment API, or deploy a new
version of the dashboard itself.

## Release sequence

1. Keep the stable Deployment serving the `traffic` Service.
2. Start a separate candidate Deployment and wait for readiness.
3. Send 20 synthetic requests to stable and 20 to candidate from an in-cluster
   sampler. Refuse promotion if baseline measurements fail.
4. Require candidate errors <=5%, p95 <=250 ms, and p95 no greater than
   max(100 ms, 3 × baseline p95). The 250 ms ceiling still applies.
   Reject incomplete/invalid measurements and unexpected serving revisions.
5. Promote by changing the traffic Service selector with JSON Patch tests for
   the exact reviewed resourceVersion and selector. Wait for endpoint convergence.
6. Send another 20 requests through the traffic Service. On regression, restore
   the prior stable selector and verify 20 successful requests from that revision.

This is **blue/green routing with synthetic candidate validation**, not weighted
production canary traffic. The lab's thresholds and sample size are illustrative;
they are not a statistical proof of production reliability or universal SLOs.
The deliberately bad application keeps readiness green to show why readiness
alone cannot establish a release's application behavior.

## Scenarios and required decisions

| Candidate | Application behavior | Required result |
|---|---|---|
| healthy | HTTP 200, low latency | Promote; verify candidate responses |
| errors | HTTP 503 despite readiness | Reject before promotion |
| slow | 400 ms responses | Reject before promotion |
| late-failure | First 20 requests succeed, later requests fail | Promote, detect failure, restore stable traffic |

After a healthy success the harness explicitly resets to stable to isolate the
next scenario. This reset is not counted as a failure rollback. A rejected
candidate never receives traffic from the traffic Service. Every scenario checks
stable traffic afterward. A final real-API test changes the Service after review
and requires the stale promotion attempt to fail without changing its selector.

## Run

Dependencies: Docker, kind, kubectl, Python 3. No AWS credentials or paid resources.
Create a dedicated cluster; the runner refuses other context names and refuses to
overwrite an existing namespace.

```sh
kind create cluster --name kubevista-release-lab --image kindest/node:v1.34.0

docker build -t kubevista-release:lab scripts/release-lab
kind load docker-image kubevista-release:lab --name kubevista-release-lab
python3 -m unittest discover -s scripts/release-lab -p 'test_*.py'
python3 scripts/release-lab/release.py --context kind-kubevista-release-lab
```

Inspect `outputs/release-lab/results.json` for error rates, p95 latency, thresholds,
serving-revision checks and recovery decisions. `stale-release.json` records the
stale-mutation check. The Safe release lab workflow uploads evidence even on failure.
Cleanup: `kind delete cluster --name kubevista-release-lab`.

## Boundaries

- Rollback restores routing to the retained stable Deployment. It does not undo a
  database migration or external side effect. Old and new versions need compatible
  schemas before this pattern can be used for a real application.
- The controller refuses to overwrite concurrent Service edits. If its rollback
  guard fails, the run fails and requires an operator to inspect the new state.
- The Python runner is a foreground lab harness. SIGKILL or machine failure can
  prevent rollback; this is not a durable release controller. The retained stable
  Deployment remains available for an operator to restore routing.
- Endpoint convergence and sampled recovery do not guarantee uninterrupted service
  for persistent client connections or every node in a production cluster.
- It uses a single-node cluster, synthetic traffic and small samples. Do not turn
  lab results into production uptime, customer impact or deployment-time claims.
