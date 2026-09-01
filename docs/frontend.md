# Frontend engineering standard

KubeVista is an operations console, not a marketing page. The first viewport is
a working surface: cluster identity, connection state, last observation time,
resource health, phase distribution, and inventory provenance. There is no hero
section, decorative gradient, fabricated time series, or invented workload row.

## Visual system

- Neutral charcoal surfaces and thin structural borders carry hierarchy.
- Blue is reserved for selection, green/yellow/red for operational state.
- Corners stay between three and five pixels; panels do not float as generic
  rounded cards.
- System and monospace fonts avoid a runtime font request and keep identifiers
  legible.
- Icons are small, purpose-built inline SVGs with a consistent stroke system.
- Every primary navigation item opens a working live-data view; no placeholder
  routes are presented as product functionality.

## Data contract and states

The frontend consumes eight read-only contracts under `/api/v1`: `summary`,
`workloads`, `network`, `events`, `observability`, `security`, `cost`, and
`settings`. Views fetch only their own data, refresh every 15 seconds, expose a
manual refresh control, and retain the last successful response when a refresh
fails. Demo mode is always labeled explicitly.

Platform-posture rows describe committed baseline configuration, not runtime
evaluation. The security view is an informational runtime scan of Pod security
contexts, not an admission controller. The cost view is a directional model,
not an AWS invoice, and lists its exclusions beside the estimate.

## View inventory

| View | Live source | Primary operator question |
| --- | --- | --- |
| Overview | Nodes, namespaces, Pods | Is the cluster healthy now? |
| Workloads | Deployments, StatefulSets, DaemonSets, Jobs, CronJobs | Which controllers are unavailable? |
| Network | Services, Ingresses, NetworkPolicies | What is exposed and isolated? |
| Events | Core Kubernetes Events | What changed or is warning? |
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
