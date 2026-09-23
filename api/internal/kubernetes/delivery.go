package kubernetes

import (
	"context"
	"encoding/json"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"time"
)

type DeliveryWorkload struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Images    []string `json:"images"`
	Desired   int32    `json:"desired"`
	Updated   int32    `json:"updated"`
	Ready     int32    `json:"ready"`
	State     string   `json:"state"`
}
type Delivery struct {
	BuildState    string             `json:"buildState"`
	BuildRevision string             `json:"buildRevision"`
	ArgoState     string             `json:"argoState"`
	Workloads     []DeliveryWorkload `json:"workloads"`
	ObservedAt    time.Time          `json:"observedAt"`
}

func (c *Client) Delivery(ctx context.Context) (Delivery, error) {
	result := Delivery{Workloads: []DeliveryWorkload{}, ObservedAt: time.Now().UTC(), BuildState: "unknown", ArgoState: "unknown"}
	list, err := c.client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=kubevista"})
	if err != nil {
		return result, err
	}
	for _, d := range list.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		item := DeliveryWorkload{Name: d.Name, Namespace: d.Namespace, Desired: desired, Updated: d.Status.UpdatedReplicas, Ready: d.Status.ReadyReplicas, Images: []string{}, State: "progressing"}
		for _, container := range d.Spec.Template.Spec.Containers {
			item.Images = append(item.Images, container.Image)
		}
		if d.Status.ObservedGeneration >= d.Generation && d.Status.UpdatedReplicas == desired && d.Status.AvailableReplicas == desired && d.Status.Replicas == desired {
			item.State = "ready"
		}
		for _, condition := range d.Status.Conditions {
			if condition.Type == "Progressing" && condition.Status == "False" && condition.Reason == "ProgressDeadlineExceeded" {
				item.State = "failed"
			}
		}
		result.Workloads = append(result.Workloads, item)
	}
	// Optional observers are read-only. Failure never becomes a success signal.
	if os.Getenv("KUBEVISTA_GITHUB_DELIVERY") == "true" {
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/andy98w/Kubernetes-Dashboard/actions/workflows/images.yml/runs?branch=main&per_page=1", nil)
		if e == nil {
			req.Header.Set("Accept", "application/vnd.github+json")
			response, e := (&http.Client{Timeout: 3 * time.Second}).Do(req)
			if e == nil {
				defer response.Body.Close()
				if response.StatusCode == http.StatusOK {
					var payload struct {
						Runs []struct {
							Status     string `json:"status"`
							Conclusion string `json:"conclusion"`
							SHA        string `json:"head_sha"`
						} `json:"workflow_runs"`
					}
					if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload) == nil && len(payload.Runs) > 0 {
						run := payload.Runs[0]
						result.BuildState = run.Status
						if run.Status == "completed" {
							result.BuildState = run.Conclusion
						}
						result.BuildRevision = run.SHA
					}
				}
			}
		}
	}
	if os.Getenv("KUBEVISTA_ARGO_DELIVERY") == "true" && c.client.Discovery().RESTClient() != nil {
		var app struct {
			Status struct {
				Sync struct {
					Status string `json:"status"`
				} `json:"sync"`
				Health struct {
					Status string `json:"status"`
				} `json:"health"`
			} `json:"status"`
		}
		raw, e := c.client.Discovery().RESTClient().Get().AbsPath("/apis/argoproj.io/v1alpha1/namespaces/argocd/applications/kubevista-dashboard").DoRaw(ctx)
		if e == nil && json.Unmarshal(raw, &app) == nil && app.Status.Sync.Status != "" {
			result.ArgoState = app.Status.Sync.Status + " / " + app.Status.Health.Status
		}
	}
	result.ObservedAt = time.Now().UTC()
	return result, nil
}
