package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type ContainerImage struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}
type PodDetail struct {
	Name       string    `json:"name"`
	Phase      string    `json:"phase"`
	Ready      int       `json:"ready"`
	Containers int       `json:"containers"`
	Restarts   int32     `json:"restarts"`
	Node       string    `json:"node"`
	CreatedAt  time.Time `json:"createdAt"`
}
type WorkloadDetail struct {
	Workload   Workload          `json:"workload"`
	Strategy   string            `json:"strategy"`
	Selector   map[string]string `json:"selector"`
	Labels     map[string]string `json:"labels"`
	Images     []ContainerImage  `json:"images"`
	Pods       []PodDetail       `json:"pods"`
	Services   []Service         `json:"services"`
	Policies   []Policy          `json:"policies"`
	Events     []Event           `json:"events"`
	ObservedAt time.Time         `json:"observedAt"`
}

type IncidentEvidence struct {
	Source string    `json:"source"`
	Detail string    `json:"detail"`
	At     time.Time `json:"at"`
}
type Incident struct {
	ID        string             `json:"id"`
	Severity  string             `json:"severity"`
	Status    string             `json:"status"`
	Title     string             `json:"title"`
	Summary   string             `json:"summary"`
	Namespace string             `json:"namespace"`
	Resource  string             `json:"resource"`
	StartedAt time.Time          `json:"startedAt"`
	Evidence  []IncidentEvidence `json:"evidence"`
}
type Incidents struct {
	Items      []Incident `json:"items"`
	ObservedAt time.Time  `json:"observedAt"`
}
type ClusterUpdate struct {
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	At        time.Time `json:"at"`
}

func (c *Client) WorkloadDetail(ctx context.Context, namespace, kind, name string) (WorkloadDetail, error) {
	all, err := c.Workloads(ctx)
	if err != nil {
		return WorkloadDetail{}, err
	}
	var selected *Workload
	for i := range all.Items {
		if strings.EqualFold(all.Items[i].Kind, kind) && all.Items[i].Namespace == namespace && all.Items[i].Name == name {
			selected = &all.Items[i]
			break
		}
	}
	if selected == nil {
		return WorkloadDetail{}, fmt.Errorf("workload %s/%s/%s not found", namespace, kind, name)
	}

	selector, templateLabels, strategy, images := map[string]string{}, map[string]string{}, "Controller managed", []ContainerImage{}
	addImages := func(spec corev1.PodSpec) {
		for _, item := range spec.Containers {
			images = append(images, ContainerImage{item.Name, item.Image})
		}
	}
	switch strings.ToLower(kind) {
	case "deployment":
		o, e := c.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return WorkloadDetail{}, e
		}
		selector = o.Spec.Selector.MatchLabels
		templateLabels = o.Spec.Template.Labels
		strategy = string(o.Spec.Strategy.Type)
		addImages(o.Spec.Template.Spec)
	case "statefulset":
		o, e := c.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return WorkloadDetail{}, e
		}
		selector = o.Spec.Selector.MatchLabels
		templateLabels = o.Spec.Template.Labels
		strategy = string(o.Spec.UpdateStrategy.Type)
		addImages(o.Spec.Template.Spec)
	case "daemonset":
		o, e := c.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return WorkloadDetail{}, e
		}
		selector = o.Spec.Selector.MatchLabels
		templateLabels = o.Spec.Template.Labels
		strategy = string(o.Spec.UpdateStrategy.Type)
		addImages(o.Spec.Template.Spec)
	case "job":
		o, e := c.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return WorkloadDetail{}, e
		}
		selector = o.Spec.Selector.MatchLabels
		templateLabels = o.Spec.Template.Labels
		strategy = "Run to completion"
		addImages(o.Spec.Template.Spec)
	case "cronjob":
		o, e := c.client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return WorkloadDetail{}, e
		}
		templateLabels = o.Spec.JobTemplate.Spec.Template.Labels
		strategy = o.Spec.Schedule
		addImages(o.Spec.JobTemplate.Spec.Template.Spec)
	}
	detail := WorkloadDetail{Workload: *selected, Strategy: strategy, Selector: selector, Labels: templateLabels, Images: images, Pods: []PodDetail{}, Services: []Service{}, Policies: []Policy{}, Events: []Event{}, ObservedAt: time.Now().UTC()}
	pods := &corev1.PodList{}
	if len(selector) > 0 {
		pods, err = c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(selector).String()})
		if err != nil {
			return WorkloadDetail{}, fmt.Errorf("list workload pods: %w", err)
		}
	}
	podNames := map[string]bool{}
	for _, pod := range pods.Items {
		ready, restarts := 0, int32(0)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCount
		}
		detail.Pods = append(detail.Pods, PodDetail{pod.Name, string(pod.Status.Phase), ready, len(pod.Spec.Containers), restarts, pod.Spec.NodeName, pod.CreationTimestamp.Time})
		podNames[pod.Name] = true
	}
	network, err := c.Network(ctx)
	if err == nil {
		for _, service := range network.Services {
			if service.Namespace != namespace {
				continue
			}
			obj, e := c.client.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{})
			if e == nil && selectorMatches(obj.Spec.Selector, templateLabels) {
				detail.Services = append(detail.Services, service)
			}
		}
		policies, e := c.client.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
		if e == nil {
			for _, p := range policies.Items {
				s, e := metav1.LabelSelectorAsSelector(&p.Spec.PodSelector)
				if e == nil && s.Matches(labels.Set(templateLabels)) {
					detail.Policies = append(detail.Policies, Policy{p.Namespace, p.Name, len(p.Spec.Ingress), len(p.Spec.Egress)})
				}
			}
		}
	}
	events, err := c.Events(ctx)
	if err == nil {
		for _, event := range events.Items {
			parts := strings.SplitN(event.Object, "/", 2)
			if event.Namespace == namespace && (event.Object == kind+"/"+name || (len(parts) == 2 && podNames[parts[1]])) {
				detail.Events = append(detail.Events, event)
			}
		}
	}
	return detail, nil
}

func selectorMatches(selector, target map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if target[k] != v {
			return false
		}
	}
	return true
}

func (c *Client) Incidents(ctx context.Context) (Incidents, error) {
	events, err := c.Events(ctx)
	if err != nil {
		return Incidents{}, err
	}
	result := Incidents{Items: []Incident{}, ObservedAt: time.Now().UTC()}
	for i, event := range events.Items {
		if event.Type != "Warning" {
			continue
		}
		severity := "Warning"
		lower := strings.ToLower(event.Reason)
		if strings.Contains(lower, "failed") || strings.Contains(lower, "backoff") || strings.Contains(lower, "unhealthy") {
			severity = "High"
		}
		result.Items = append(result.Items, Incident{fmt.Sprintf("evt-%d", i+1), severity, "Active", event.Reason + " on " + event.Object, event.Message, event.Namespace, event.Object, event.LastSeen, []IncidentEvidence{{"Kubernetes event", fmt.Sprintf("%s (count %d)", event.Message, event.Count), event.LastSeen}}})
	}
	return result, nil
}

func (c *Client) Updates(ctx context.Context) (<-chan ClusterUpdate, error) {
	watcher, err := c.client.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("watch pods: %w", err)
	}
	updates := make(chan ClusterUpdate, 8)
	go func() {
		defer close(updates)
		defer watcher.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}
				accessor, err := meta.Accessor(event.Object)
				if err != nil {
					continue
				}
				select {
				case updates <- ClusterUpdate{"pods", strings.ToLower(string(event.Type)), accessor.GetNamespace(), accessor.GetName(), time.Now().UTC()}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return updates, nil
}
