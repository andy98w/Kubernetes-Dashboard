# Habitat prototype

The station is the default landing page. Choose **Tables** for dense inventory. Robots represent
observed pods grouped by their reported node assignment; unscheduled pods appear
in a waiting bay. Clicking a robot opens the existing workload drawer, including
evidence and guarded operations. Search and namespace filters apply to both views.

Ready pods have mint lights, unready/failed pods amber repair indicators,
unscheduled pods a waiting animation, and completed pods a resting state. These
are coarse states from pod phase and readiness—not diagnoses of a specific crash.
Idle bobbing and blinking are decorative, not CPU utilization or traffic signals.
There are no invented pod migrations. Node rooms do not measure node capacity or
show empty nodes, because this prototype reads workload details rather than a full
node topology endpoint.

The view loads up to 24 matching workload details with four concurrent requests,
ten-second per-request timeouts, abort on unmount/filter change, and explicit
partial-failure messaging. Pods shared by multiple controller inventories are
deduplicated by namespace/name. Each room renders up to nine robots;
additional pods are paginated. Parent inventory refreshes update assignments.
The previous observation remains visible and labeled during refresh.

SVG artwork and CSS transforms require no animation or 3D dependency. Pause motion
and OS reduced-motion preferences stop animation. Buttons have keyboard focus and
accessible pod/state labels. Demo mode explicitly labels synthetic assignments.

Verified locally: TypeScript/production build; narrow-layout browser rendering;
simulated readiness failure changes its robot to Needs attention; clicking it opens
the matching failure evidence and operations drawer. Live-cluster placement and
rollout animation have not been tested. This is the first visual prototype, not a
replacement for the inventory tables or a continuous event-stream visualization.

## Connected station

The second prototype replaces cards with isometric cutaway rooms connected by
decorative corridors. Robots take small idle steps within their own room. Drag
empty space to pan; use plus/minus and Fit to control the camera. Rooms and robots
are keyboard-focusable, and hover/focus reveals details in the inspection panel.
Clicking a bot still opens the existing workload drawer. Use `?habitat=1#/workloads`
to open directly in Habitat mode (the query is now optional). The table view stays available.

- **Room lights:** amber means one or more observed pods are unready/failed. This
  is not a claim of a confirmed incident, unhealthy node, or firing Prometheus alert.
- **Network:** toggle dashed connections from the Service hub to rooms containing
  workloads with matched Services. The hub reveals the Service names. This is
  a coarse relationship overlay, not actual traffic, endpoint health, routing, or
  NetworkPolicy enforcement. Corridors never represent network connectivity.
- **Events:** the optional log lists eight recent observations for loaded workloads,
  with timestamps and drilldown. Demo lab timeline entries are labeled as simulated.
  Warning events have an amber icon; historical events do not change room lighting.
- **Telemetry:** the terminal opens the existing Observability page. Bot inspection
  offers Load logs & metrics, backed by the existing Prometheus integration when
  configured. Grafana is not embedded and no utilization readings are fabricated.

Habitat hides summary metric cards and verbose inventory descriptions. Most copy
lives in hover/focus details or the expandable map key. The scene uses SVG/CSS
rather than a game engine; it adds no third-party rendering dependency. Tables
remain the preferred dense inventory and accessible fallback for large clusters.

## Station-first navigation

The root URL, `#/station`, `#/workloads`, and the legacy `#/overview` link now open
the station. No query flag is needed. The large dashboard sidebar is replaced by
a tools dock. Health (`#/summary`), network, events, incidents, telemetry,
security, cost, and settings open in inspection panels over the persistent map.
Their existing observations and tables are retained; they are no longer separate
landing dashboards. Deep links, browser history, and scene terminal links use the
same panels. Close, Escape, or the backdrop returns to the station without clearing
the search, namespace filter, camera, or motion settings. Background controls are
inert while an inspection is open and Tab remains in the inspection.

Workload data refreshes independently of whether a panel's request succeeds. Failed
requests are labeled and never replaced by demo values. Demo inspection panels
retain their demo label. The simulation controls live under **Simulation lab**.

Verification: frontend build and browser checks for the root default, network
inspection, Escape dismissal, and retained search filter. Playwright regression
specs cover the default route aliases, deep-linked inspection, and filter retention;
the existing table/incident tests now explicitly select Tables and open the lab.
# Single-window controls

## Topology detail

Facility rooms now contain named component consoles. Selecting a room opens
its topology detail: configured OTel pipelines and named monitoring tools,
managed control-plane responsibilities and the Terraform subnet plan, or
loaded Service-selector → pod → node relationships. These are not packet traces.
Telemetry tool selection highlights matching loaded workloads. Components
absent from the loaded subset remain labeled Configured, not falsely absent
from the cluster. node-exporter and kube-state-metrics are listed only when
matching loaded workloads exist.

The station namespace dropdown is replaced by colored namespace highlights;
pods stay in their physical node rooms. Tables retain namespace filtering.
Worker rooms use two columns with larger floors. The map still caps workload
detail loading at 24 and pages pods in groups of nine. It is not a complete
cluster inventory: verified per-Service endpoint membership, live AWS subnet
mapping, and individual managed control-plane internals remain unavailable.

## Facility room health

Control plane, Services, and Telemetry are now separate rooms. In live mode,
`GET /api/v1/station-health` probes API-server `/readyz`, explicitly ready
EndpointSlice endpoints for selector-backed Services, and telemetry workload
replica readiness. These are scoped readiness indicators, not end-to-end traffic,
ingestion, query, scheduler, or controller-manager health checks. Selectorless
Services are excluded. Missing permissions or observations produce Unknown.
The reader Helm roles permit only GET on the non-resource `/readyz` endpoint.

The browser polls every 15 seconds and expires observations after 45 seconds;
failed fetches clear previous health. Demo mode uses explicitly labeled simulated
signals. No live cluster deployment was performed for this change. Click rooms
to open the related health, network, or observability inspection.

The station fills the viewport; tables and inspection panels scroll internally.
The left navigation rail expands to show labels. Mouse-wheel and trackpad scrolling
over the map zoom around the pointer (0.45×–3×); drag empty space to pan.
Focus the map to use `+`, `-`, or `0` to reset the camera. Wheel handling is scoped
to the SVG, so event lists and inspection panels keep their normal scrolling.

The shared `Select` component retains native keyboard and mobile picker behavior.
Popup entry motion respects reduced-motion preferences. The robot logo and room
consoles are local SVG assets; decorative instruments do not imply live metrics.
