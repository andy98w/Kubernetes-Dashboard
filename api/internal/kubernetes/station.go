package kubernetes

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type RoomHealth struct {
	State  string `json:"state"`
	Detail string `json:"detail"`
}
type StationHealth struct {
	Control    RoomHealth `json:"control"`
	Services   RoomHealth `json:"services"`
	Telemetry  RoomHealth `json:"telemetry"`
	ObservedAt time.Time  `json:"observedAt"`
}

// Indicators describe specific probes, never inferred end-to-end health.
func (c *Client) StationHealth(ctx context.Context) (StationHealth, error) {
	unknown := RoomHealth{"unknown", "Probe unavailable or permission denied"}
	result := StationHealth{unknown, unknown, unknown, time.Now().UTC()}
	if rest := c.client.Discovery().RESTClient(); rest != nil {
		var code int
		err := rest.Get().AbsPath("/readyz").Do(ctx).StatusCode(&code).Error()
		if err == nil && code == 200 {
			result.Control = RoomHealth{"ready", "API server /readyz passed. Scheduler and controller health are not independently measured."}
		} else if code >= 500 {
			result.Control = RoomHealth{"warning", "API server readiness probe failed"}
		}
	}
	services, err := c.client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err == nil {
		slices, sliceErr := c.client.DiscoveryV1().EndpointSlices("").List(ctx, metav1.ListOptions{})
		if sliceErr == nil {
			ready := map[string]bool{}
			known := map[string]bool{}
			for _, slice := range slices.Items {
				key := slice.Namespace + "/" + slice.Labels["kubernetes.io/service-name"]
				for _, ep := range slice.Endpoints {
					if ep.Conditions.Ready != nil {
						known[key] = true
						if *ep.Conditions.Ready && len(ep.Addresses) > 0 {
							ready[key] = true
						}
					}
				}
			}
			eligible, missing, uncertain := 0, 0, 0
			for _, svc := range services.Items {
				if len(svc.Spec.Selector) == 0 {
					continue
				}
				eligible++
				key := svc.Namespace + "/" + svc.Name
				if !known[key] {
					uncertain++
				} else if !ready[key] {
					missing++
				}
			}
			if missing > 0 {
				result.Services = RoomHealth{"warning", "At least one selector-backed Service has no explicitly ready endpoint"}
			} else if eligible > 0 && uncertain == 0 {
				result.Services = RoomHealth{"ready", "All selector-backed Services have a ready endpoint. This is not a traffic probe."}
			} else {
				result.Services = RoomHealth{"unknown", "No eligible Services or some endpoint readiness is unknown"}
			}
		}
	}
	obs, err := c.Observability(ctx)
	if err == nil && len(obs.Components) > 0 {
		result.Telemetry = RoomHealth{"ready", "Observed telemetry workload replicas are ready; ingestion and query success are not tested."}
		for _, component := range obs.Components {
			if component.Desired == 0 {
				result.Telemetry = RoomHealth{"unknown", "A telemetry component is scaled to zero"}
			}
			if component.Ready < component.Desired {
				result.Telemetry = RoomHealth{"warning", "A telemetry workload has unavailable replicas"}
				break
			}
		}
	}
	result.ObservedAt = time.Now().UTC()
	return result, nil
}
