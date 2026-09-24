# September 23 audit follow-up

## Implemented

- Updated stale browser-test labels and added the error-grouping unit tests to CI.
- Unknown facility health now stays unknown visually and in accessible labels.
  Fixture mode does not imply successful live health probes.
- Replaced the live map's 24-workload cap and per-workload fetch fan-out with a
  station inventory endpoint. It uses a fixed number of resource lists plus
  paginated pod reads, checks controller UIDs, preserves current-owner events,
  and reports unmatched pods rather than guessing ownership. This is still a
  polling snapshot, not an informer-backed cache or atomic cluster snapshot.
- Reduced external connection clutter: dependency-to-egress lines are subdued;
  selecting a dependency reveals its floor route to matching workloads.
- Moved the error desk beside telemetry, improved label sizes, and reserved a
  control strip so dependency objects are not hidden under namespace chips.
- Added credential-gated local investigation; custom headers alone no longer
  authorize evidence access. Production remains disabled.
- Added the [local error service](local-error-service.md): actual browser error
  category capture, file persistence, bounded ingestion, grouping, version-checked
  resolution, and regression on recurrence. Existing fixture examples remain
  separate from persisted issues.

## Verification and limits

Go package tests, including the race detector, cover current-owner event matching,
more than 24 workloads, stale controller rejection, credential checks, persistence,
version conflicts, regression, ingestion limits, and failed storage writes.
Frontend type checking, normal/fixture builds, and error-grouping unit tests pass.

Connected Chrome verification exercised actual browser error → saved issue →
resolve → recurrence. The count and Regressed state survived restarting the API
and frontend. The map's top-row overlap was checked visually after the fix.

The Playwright suite could not launch Chromium in this macOS sandbox: Mach port
registration was denied before any tests ran. Installing Chromium did not resolve
that restriction. The full suite must still pass on an unrestricted machine or CI.
The working tree also has a local Go cache under `api/work`; package checks here
use `./cmd/... ./internal/...` to exclude that non-source directory.

No EKS apply, application migration, production credential change, Git commit, or
push was performed. The fleet remains fixture-backed. The local issue service is
not HA and does not replace Sentry's production access control, source maps,
notifications, retention, or trace correlation. Those are separate delivery work,
not features validated by this pass.
