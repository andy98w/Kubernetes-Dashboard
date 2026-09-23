# Durable operation controller

The original operations path kept plans in one API process. This opt-in path
stores reviewed intent in Kubernetes and finishes approved work after that process
restarts. It covers Deployment scaling, restart, and rollback. It does not add
automatic remediation or let the AI investigator approve changes.

## State and ownership

`Planned → Approved → Executing → Verifying → Succeeded`

`Expired`, `Failed`, and `NeedsReview` are terminal alternatives. A receipt saying
`Approved` or `Verifying` is not a claim that the workload recovered. The browser
polls receipts every three seconds while the operations panel is open.

Each plan is a labeled ConfigMap in a separate operation-store namespace. It
contains the exact reviewed target UID and resourceVersion, desired replicas and
pod template, approval actor, timestamps, claim owner, claim epoch, and state
history. Restart timestamps are chosen once, not regenerated on retries. Rollback
stores the reviewed template rather than looking up a possibly different revision
at execution time. Records survive API restarts, not loss of the backing cluster.

Approval uses a resourceVersion-conditional ConfigMap update. Replaying the same
request as the same actor returns the existing receipt; changing the request is
rejected. An approval race can return a conflict: retry the same plan, not a newly
invented operation. Unapproved plans expire after five minutes.

Workers claim a record for 15 seconds with a unique process incarnation and an
increasing epoch. Every claim and receipt update uses Kubernetes optimistic
concurrency. An old copy cannot overwrite a newer owner's receipt. Active work
is revisited about every three seconds, renewing the claim. API failures use the
work queue's rate-limited retry path.

## Partial failures and fencing

The Deployment patch tests **both UID and the original reviewed resourceVersion**
before atomically setting the operation marker and the desired change. It never
refreshes that precondition to make a retry succeed. This also means an unrelated
status update can invalidate a plan: conservative, but inconvenient on a busy
Deployment. A new plan is required instead of silently accepting drift.

If Kubernetes accepts the patch and the worker disappears before saving its
receipt, the next worker reads the target. A matching operation marker and desired
intent mean it can verify the existing rollout without restarting it again.
Changed UID, conflicting intent, missing marker after application, or a superseding
change lead to `NeedsReview`. Transient errors retry. A final admission rejection
is `Failed`. Work that has not established recovery within ten minutes of approval
goes to review when the API becomes reachable again. No automatic inverse action
is attempted: undoing an uncertain change can overwrite somebody else's work.

This is not an exactly-once distributed transaction. Lease expiry alone cannot
stop an HTTP request already in flight. A stale worker might still submit the
same approved intent if the target has not changed. The target's immutable
preconditions allow at most one successful effect for that plan; ConfigMap CAS
protects its receipt. The epoch is **not** a fencing token understood by arbitrary
external services. Extending this controller to payments or another API needs
that destination's own idempotency/concurrency mechanism.

## Election versus active-active

Default durable mode uses a Kubernetes Lease to elect one worker process; all
API replicas can create and approve plans. Lease duration is 15 seconds, renew
deadline 10 seconds, retry period 2 seconds. Losing leadership stops the worker
and the API process so Kubernetes can restart it with a new incarnation. The lease
is not released early while requests could still be in flight.

`operations.activeActive=true` runs workers on every replica. They still coordinate
per record through claims and CAS. This removes the *global* leader, not all
coordination. Two replicas can observe the same event, but cannot safely commit
different effects from the same reviewed target version. There is no horizontal
throughput benchmark yet; each process currently runs one queue worker.

ConfigMap and Deployment shared informers feed a typed, rate-limited work queue.
An index maps changed Deployments to their operation records. Startup lists recover
pending records even when no new watch event occurs. Delayed reconciliation covers
lease expiry and timeouts. Cached objects schedule work; fresh API reads make
decisions. Duplicate or coalesced watch events are normal.

## Local use

The feature is off by default. For the existing local Helm lab, add these options
to the normal install command, using images built from this checkout:

```sh
helm upgrade --install kubevista platform/apps/kubevista-local \
  --namespace kubevista-system --create-namespace \
  --set operations.enabled=true --set operations.durable=true \
  --set apiReplicas=2
```

For the active-active comparison, also set `--set operations.activeActive=true`.
The default allowed target namespace is `kubevista-lab`. Follow the existing
[incident lab](incident-lab.md) for images, workloads, and local access. No AWS
deployment is needed. Do not expose the local chart as an authenticated public
service; production authentication remains a separate requirement.

Direct API configuration uses `KUBEVISTA_DURABLE_OPERATIONS=true`,
`KUBEVISTA_OPERATIONS_ENABLED=true`, `KUBEVISTA_OPERATION_STORE_NAMESPACE`, and
optionally `KUBEVISTA_CONTROLLER_ACTIVE_ACTIVE=true` and `KUBEVISTA_CONTROLLER_ID`.
Durable mode rejects demo mode and disabled operations. Every API replica must use
the same store and compatible namespace/replica policies. The production chart is
not automatically switched to this experimental backend.

## Security and operational limits

- The dedicated Role grants ConfigMap get/list/watch/create/update and Lease
  get/create/update. It grants no delete permission. Target Deployment permissions
  remain namespace-scoped in the existing operator role.
- ConfigMaps contain pod templates, which can include **literal environment
  secrets**. They are not a secret vault. Restrict readers/writers, configure
  appropriate encryption at rest, and prefer Secret references in workloads.
  The HTTP receipt endpoint does not expose raw templates.
- Anyone able to edit operation records is trusted as an operator. This is not
  a tamper-proof audit ledger; state history is mutable by that identity.
- The 256 KiB record bound limits template size. No retention controller exists.
  The API returns the latest 50 receipts but older records remain stored. Plan
  creation needs access control/rate limiting before public multi-tenant use.
- The store namespace is marked `helm.sh/resource-policy: keep`. Uninstalling
  leaves records behind. Review/export and explicitly remove that namespace only
  when its history is no longer needed. Disabling durable mode leaves approved
  records pending; re-enabling can resume eligible work. Do not mix memory and
  durable replicas behind one service.
- Admission mutation can change the applied template; a mismatch is flagged for
  review rather than accepted as success. GitOps/HPA changes can invalidate plans.
  There is no force-rebase, cancellation, or automatic conflict resolution.

## Reproduce the failure tests

```sh
bash scripts/test-durable-operations.sh
```

The script downloads SHA-512-pinned envtest v1.36.2 binaries for Apple Silicon or
Linux amd64 and runs race-enabled tests against a real loopback kube-apiserver and
etcd. It never reads kubeconfig. Child processes and temporary data are cleaned up
by the test harness. Downloaded binaries remain under ignored `work/` for inspection.
CI has a separate job so a skipped optional local test cannot masquerade as this
integration check.

Coverage includes concurrent/replayed approvals, an injected interruption between
target write and receipt persistence, stale receipt and target writes, competing
plans, target drift/recreation, plan expiry, unapproved work, pinned rollback
templates, two active workers, and leader takeover. Recovery status is explicitly
supplied by the test: envtest has no kubelet or Deployment controller. These tests
prove API concurrency and controller behavior, **not real pod recovery or EKS
availability**. The earlier EKS resilience evidence remains a separate experiment.

Implementation: `api/internal/kubernetes/durable_operations.go` and
`operation_controller.go`; integration checks: `durable_integration_test.go`.
See [Kubernetes API concurrency](https://kubernetes.io/docs/reference/using-api/api-concepts/)
and [client-go's leader-election caveat](https://pkg.go.dev/k8s.io/client-go/tools/leaderelection).

### Local verification — September 22, 2026

- All eleven real-API scenarios passed with `-race`, including a second run through
  the download/checksum/test script. The full Go package test suite and `go vet`
  also passed (explicit `./internal/... ./cmd/...` paths avoid a local nested Go
  module cache under `api/work/`).
- Frontend TypeScript/build and local Helm lint/render checks passed.
- The in-app browser completed a simulated failure → reviewed rollback → receipt
  → explicit recovery check. This checks the existing UI flow, not a live rollout.
- The existing 20-case desktop/mobile Playwright suite could not launch Chromium
  inside the macOS sandbox (`MachPortRendezvousServer` permission denied). It did
  not reach application assertions. CI still runs that suite; no CI result is
  claimed for these unpushed changes.
