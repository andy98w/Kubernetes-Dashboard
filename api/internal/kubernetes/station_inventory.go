package kubernetes

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

type StationResident struct {
	PodDetail
	Workload Workload `json:"workload"`
	Services []string `json:"services"`
}
type StationInventory struct {
	Residents     []StationResident    `json:"residents"`
	Events        []StationObservation `json:"events"`
	UnmatchedPods int                  `json:"unmatchedPods"`
	ObservedAt    time.Time            `json:"observedAt"`
}
type StationObservation struct {
	Event
	Workload Workload `json:"workload"`
}

// Collect once per resource type, rather than relisting all controllers per workload.
// UID ownership (not coincidentally matching labels) associates pods to controllers.
func (c *Client) StationInventory(ctx context.Context) (StationInventory, error) {
	result := StationInventory{Residents: []StationResident{}, ObservedAt: time.Now().UTC()}
	all, err := c.Workloads(ctx)
	if err != nil {
		return result, err
	}
	controllers := map[string]Workload{}
	for _, w := range all.Items {
		controllers[w.Namespace+"/"+w.Kind+"/"+w.Name] = w
	}
	// Resolve current controller identities, so old pods cannot attach to a
	// replacement controller that happens to reuse the same namespace/name.
	identities := map[string]types.UID{}
	deployments, err := c.client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, v := range deployments.Items {
		identities[v.Namespace+"/Deployment/"+v.Name] = v.UID
	}
	stateful, err := c.client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, v := range stateful.Items {
		identities[v.Namespace+"/StatefulSet/"+v.Name] = v.UID
	}
	daemons, err := c.client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, v := range daemons.Items {
		identities[v.Namespace+"/DaemonSet/"+v.Name] = v.UID
	}
	jobs, err := c.client.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, v := range jobs.Items {
		identities[v.Namespace+"/Job/"+v.Name] = v.UID
	}
	ownerWorkloads := map[types.UID]Workload{}
	for key, uid := range identities {
		if uid != "" {
			ownerWorkloads[uid] = controllers[key]
		}
	}
	sets, err := c.client.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	rsOwners := map[string]string{}
	for _, rs := range sets.Items {
		if o := metav1.GetControllerOf(&rs); o != nil && o.Kind == "Deployment" {
			key := rs.Namespace + "/Deployment/" + o.Name
			if o.UID != "" && identities[key] == o.UID {
				rsOwners[string(rs.UID)] = key
				ownerWorkloads[rs.UID] = controllers[key]
			}
		}
	}
	services, err := c.client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	continuation := ""
	for {
		pods, e := c.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{Limit: 500, Continue: continuation})
		if e != nil {
			return result, fmt.Errorf("list station pods: %w", e)
		}
		for _, p := range pods.Items {
			o := metav1.GetControllerOf(&p)
			key := ""
			if o != nil {
				key = p.Namespace + "/" + o.Kind + "/" + o.Name
				if o.Kind == "ReplicaSet" {
					key = rsOwners[string(o.UID)]
				} else if o.UID == "" || identities[key] != o.UID {
					key = ""
				}
			}
			w, ok := controllers[key]
			if !ok {
				result.UnmatchedPods++
				continue
			}
			if p.UID != "" {
				ownerWorkloads[p.UID] = w
			}
			ready := 0
			var restarts int32
			for _, cs := range p.Status.ContainerStatuses {
				if cs.Ready {
					ready++
				}
				restarts += cs.RestartCount
			}
			matched := []string{}
			for _, s := range services.Items {
				if s.Namespace == p.Namespace && len(s.Spec.Selector) > 0 && labels.SelectorFromSet(s.Spec.Selector).Matches(labels.Set(p.Labels)) {
					matched = append(matched, s.Namespace+"/"+s.Name)
				}
			}
			result.Residents = append(result.Residents, StationResident{PodDetail{p.Name, string(p.Status.Phase), ready, len(p.Spec.Containers), restarts, p.Spec.NodeName, p.CreationTimestamp.Time}, w, matched})
		}
		continuation = pods.Continue
		if continuation == "" {
			break
		}
	}
	result.Events = []StationObservation{}
	events, err := c.client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, e := range events.Items {
		w, ok := ownerWorkloads[e.InvolvedObject.UID]
		if !ok || w.Name == "" {
			continue
		}
		seen := e.LastTimestamp.Time
		if seen.IsZero() {
			seen = e.EventTime.Time
		}
		if seen.IsZero() {
			seen = e.CreationTimestamp.Time
		}
		result.Events = append(result.Events, StationObservation{Event{e.Type, e.Reason, e.Namespace, e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name, e.Message, e.Count, seen}, w})
	}
	return result, nil
}
