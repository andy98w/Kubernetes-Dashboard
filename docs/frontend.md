# Frontend notes

The first screen answers a few basic questions: which cluster is selected, when
the data was collected, whether the API is connected, and whether any nodes or
pods need attention. The rest of the navigation follows the resources exposed
by the Go API.

## Visual system

- The UI uses dark neutral surfaces with blue for selection and
  green/yellow/red for status.
- System fonts avoid an extra network request. Resource names and IDs use a
  monospace stack.
- The icons are inline SVGs, so the frontend has no icon package or runtime
  asset request.
- Every navigation item has a working view. Unimplemented ideas stay out of the
  menu.

## Data contract and states

The frontend consumes the following read-only contracts under `/api/v1`: `summary`,
`workloads`, workload detail, `network`, `events`, `incidents`, `observability`,
`security`, `cost`, and `settings`. In cluster mode, `/api/v1/stream` carries
server-sent Pod updates from a Kubernetes watch. The browser debounces those
signals and refreshes only the active view; the 15-second poll remains as a
recovery path. Manual refresh and last-known-good data are preserved.

Selecting a workload opens a keyboard-accessible panel containing its Pods,
images, Services, matching NetworkPolicies, and recent Events. The incident
view groups related warning Events on a timeline.

The platform-posture rows come from configuration; they are not runtime policy
results. The security page checks Pod security contexts but does not enforce
admission policy. Cost uses fixed hourly rates and node labels, so it should be
read as an estimate rather than an AWS bill.

## View inventory

| View | Live source | Primary operator question |
| --- | --- | --- |
| Overview | Nodes, namespaces, Pods | Is the cluster healthy now? |
| Workloads | Deployments, StatefulSets, DaemonSets, Jobs, CronJobs | Which controllers are unavailable? |
| Network | Services, Ingresses, NetworkPolicies | What is exposed and isolated? |
| Events | Core Kubernetes Events | What changed or is warning? |
| Incidents | Correlated warning Events | What happened, what was affected, and did it recover? |
| Observability | Workloads in `observability` | Are metrics, logs, and traces available? |
| Security | Pod security contexts and NetworkPolicies | Which runtime configurations need review? |
| Cost | Node labels plus documented fixed AWS rates | What is the approximate hourly run rate? |
| Settings | Application runtime configuration | Which environment and safety mode is active? |

## Accessibility and responsive behavior

The shell uses semantic navigation, headings, articles, status regions, and
ARIA-labeled tables. Every actionable control has a keyboard focus indicator.
At narrower breakpoints metrics move from four columns to two and then one;
the desktop sidebar becomes a compact product header. Browser automation checks
desktop and mobile widths for horizontal overflow, visible headings, usable
refresh controls, accessible DOM structure, and console errors.

## Deployment boundary

Vite produces hashed static assets. An unprivileged NGINX container serves them
on port 8080, applies CSP and browser-hardening headers, caches only hashed
assets immutably, and forwards `/api` and `/healthz` to the internal Go Service.
The container runs as UID/GID 101 with a read-only root filesystem and no Linux
capabilities in Kubernetes.

## Static demo mode

`npm run build:demo` sets `VITE_DATA_MODE=demo` and bundles representative data
from the August 31 EKS run. It does not need the API, AWS credentials, or a live
cluster. The banner identifies it as a demo and links back to the deployment
records. Workload drill-down and the disruption timeline still work in this
build.

The root `vercel.json` selects this build, serves `web/dist`, and applies
browser-hardening headers. The live container build continues to use the normal
`npm run build` command and the internal Go API.
