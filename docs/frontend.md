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
- Unimplemented navigation is visibly disabled and labeled `Soon`.

## Data contract and states

`GET /api/v1/summary` is the only live frontend contract today. The UI derives
totals and percentages from its node, namespace, and pod-phase fields. Demo mode
is labeled explicitly. During initial load, counters do not pretend to be real;
after a failed refresh, the UI reports the error and retains only the last
successful response. Automatic refresh runs every 15 seconds and operators can
retry manually.

Platform-posture rows describe committed baseline configuration, not runtime
evaluation. The explanatory note makes that boundary explicit. Runtime posture,
workload detail, events, networking, observability, and cost are roadmap items.

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
