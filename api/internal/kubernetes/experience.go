package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
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
	UID        string             `json:"uid,omitempty"`
	Timeline   []IncidentEvidence `json:"timeline"`
	Recovery   *Recovery          `json:"recovery,omitempty"`
	Warnings   []string           `json:"warnings"`
	Logs       []LogExcerpt       `json:"logs"`
	Metrics    []MetricSample     `json:"metrics"`
	Diagnoses  []Diagnosis        `json:"diagnoses"`
	Workload   Workload           `json:"workload"`
	Strategy   string             `json:"strategy"`
	Selector   map[string]string  `json:"selector"`
	Labels     map[string]string  `json:"labels"`
	Images     []ContainerImage   `json:"images"`
	Pods       []PodDetail        `json:"pods"`
	Services   []Service          `json:"services"`
	Policies   []Policy           `json:"policies"`
	Events     []Event            `json:"events"`
	ObservedAt time.Time          `json:"observedAt"`
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
	var deployment *appsv1.Deployment
	var fullSelector *metav1.LabelSelector
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
		deployment, fullSelector = o, o.Spec.Selector
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
	detail.Warnings = []string{}
	detail.Timeline = []IncidentEvidence{}
	owners := map[string]bool{}
	if deployment != nil {
		detail.UID = string(deployment.UID)
		r := deploymentRecovery(deployment)
		detail.Recovery = &r
		sets, e := c.client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
		if e != nil {
			return WorkloadDetail{}, fmt.Errorf("list deployment ReplicaSets: %w", e)
		}
		for _, rs := range sets.Items {
			owner := metav1.GetControllerOf(&rs)
			if owner != nil && owner.UID == deployment.UID && owner.Kind == "Deployment" {
				owners[string(rs.UID)] = true
				detail.Timeline = append(detail.Timeline, IncidentEvidence{"ReplicaSet", rs.Name + " created (revision " + rs.Annotations["deployment.kubernetes.io/revision"] + ")", rs.CreationTimestamp.Time})
			}
		}
		for _, condition := range deployment.Status.Conditions {
			detail.Timeline = append(detail.Timeline, IncidentEvidence{"Deployment condition", string(condition.Type) + "=" + string(condition.Status) + ": " + condition.Reason + " — " + condition.Message, condition.LastUpdateTime.Time})
		}
		if r.Status == "Stalled" {
			detail.Diagnoses = append(detail.Diagnoses, Diagnosis{"RolloutStalled", "Deployment/" + name, r.Detail, "Inspect current pod failures and compare the previous revision before reviewing a rollback."})
		}
	}
	pods := &corev1.PodList{}
	if len(selector) > 0 || fullSelector != nil {
		query := labels.SelectorFromSet(selector)
		if fullSelector != nil {
			query, err = metav1.LabelSelectorAsSelector(fullSelector)
			if err != nil {
				return WorkloadDetail{}, err
			}
		}
		pods, err = c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: query.String()})
		if err != nil {
			return WorkloadDetail{}, fmt.Errorf("list workload pods: %w", err)
		}
	}
	if deployment != nil && deployment.UID != "" {
		owned := []corev1.Pod{}
		for _, pod := range pods.Items {
			owner := metav1.GetControllerOf(&pod)
			if owner != nil && owner.Kind == "ReplicaSet" && owners[string(owner.UID)] {
				owned = append(owned, pod)
			}
		}
		pods.Items = owned
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
		for _, condition := range pod.Status.Conditions {
			if !condition.LastTransitionTime.IsZero() {
				detail.Timeline = append(detail.Timeline, IncidentEvidence{"Pod condition", pod.Name + ": " + string(condition.Type) + "=" + string(condition.Status) + " " + condition.Message, condition.LastTransitionTime.Time})
			}
		}
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
				slices, err := c.client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{"kubernetes.io/service-name": service.Name}.String()})
				if err != nil {
					detail.Warnings = append(detail.Warnings, "EndpointSlices unavailable for "+service.Name)
				} else {
					ready := 0
					for _, slice := range slices.Items {
						for _, endpoint := range slice.Endpoints {
							if endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready {
								ready += len(endpoint.Addresses)
							}
						}
					}
					if ready == 0 {
						detail.Diagnoses = append(detail.Diagnoses, Diagnosis{"NoReadyEndpoints", "Service/" + service.Name, "No ready endpoint addresses were observed.", "Compare the Service selector and pod readiness. A Running pod may still fail readiness probes."})
					}
				}
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
	if err != nil {
		detail.Warnings = append(detail.Warnings, "Network inventory unavailable")
	}
	events, err := c.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, event := range events.Items {
			ref := event.InvolvedObject
			matches := strings.EqualFold(ref.Kind, kind) && ref.Name == name
			if deployment != nil && deployment.UID != "" {
				matches = matches && ref.UID == deployment.UID
			}
			podMatch := ref.Kind == "Pod" && podNames[ref.Name]
			if podMatch && ref.UID != "" {
				podMatch = false
				for _, pod := range pods.Items {
					if pod.UID == ref.UID {
						podMatch = true
					}
				}
			}
			if matches || podMatch || (ref.Kind == "ReplicaSet" && owners[string(ref.UID)]) {
				at := event.LastTimestamp.Time
				if at.IsZero() {
					at = event.EventTime.Time
				}
				if at.IsZero() {
					at = event.CreationTimestamp.Time
				}
				detail.Events = append(detail.Events, Event{event.Type, event.Reason, namespace, ref.Kind + "/" + ref.Name, event.Message, event.Count, at})
				detail.Timeline = append(detail.Timeline, IncidentEvidence{"Kubernetes event", ref.Kind + "/" + ref.Name + ": " + event.Reason + " — " + event.Message, at})
			}
		}
	} else {
		detail.Warnings = append(detail.Warnings, "Events unavailable; diagnosis is incomplete")
	}
	detail.Diagnoses = append(detail.Diagnoses, diagnosePods(pods.Items)...)
	detail.Diagnoses = append(detail.Diagnoses, diagnoseEvents(detail.Events)...)
	if detail.Diagnoses == nil {
		detail.Diagnoses = []Diagnosis{}
	}
	receipts, _ := c.RecentOperations(ctx)
	for _, record := range receipts {
		if record.Target == namespace+"/Deployment/"+name {
			detail.Timeline = append(detail.Timeline, IncidentEvidence{"Operator action", record.Action + " by " + record.Actor + ": " + record.Reason + " (" + record.Status + ")", record.CreatedAt})
		}
	}
	detail.Timeline = append(detail.Timeline, IncidentEvidence{"Observation", fmt.Sprintf("%d/%d replicas ready", selected.Ready, selected.Desired), detail.ObservedAt})
	sort.SliceStable(detail.Timeline, func(i, j int) bool { return detail.Timeline[i].At.Before(detail.Timeline[j].At) })
	sort.SliceStable(detail.Events, func(i, j int) bool { return detail.Events[i].LastSeen.After(detail.Events[j].LastSeen) })
	if len(detail.Timeline) > 100 {
		detail.Timeline = detail.Timeline[len(detail.Timeline)-100:]
	}
	if len(detail.Events) > 100 {
		detail.Events = detail.Events[:100]
	}
	detail.Metrics = []MetricSample{}
	detail.Logs = []LogExcerpt{}
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
	workloads, err := c.Workloads(ctx)
	if err != nil {
		return Incidents{}, err
	}
	result := Incidents{Items: []Incident{}, ObservedAt: time.Now().UTC()}
	for _, w := range workloads.Items {
		if w.Kind != "Deployment" || w.Status == "Healthy" {
			continue
		}
		if len(result.Items) >= 20 {
			break
		}
		detail, e := c.WorkloadDetail(ctx, w.Namespace, w.Kind, w.Name)
		if e != nil {
			return Incidents{}, e
		}
		title := "Deployment needs attention"
		if len(detail.Diagnoses) > 0 {
			title = detail.Diagnoses[0].Code
		}
		result.Items = append(result.Items, Incident{
			ID: w.Namespace + "/" + w.Name, Severity: "Warning", Status: "Active", Title: title + " on " + w.Name,
			Summary:   "Correlated controller, pod, event, and operator observations. Temporal correlation does not establish a root cause.",
			Namespace: w.Namespace, Resource: "Deployment/" + w.Name, StartedAt: w.CreatedAt, Evidence: detail.Timeline,
		})
	}
	if len(result.Items) > 0 {
		return result, nil
	}
	events, err := c.Events(ctx)
	if err != nil {
		return Incidents{}, err
	}
	for i, event := range events.Items {
		if event.Type != "Warning" {
			continue
		}
		severity := "Warning"
		lower := strings.ToLower(event.Reason)
		if strings.Contains(lower, "failed") || strings.Contains(lower, "backoff") || strings.Contains(lower, "unhealthy") {
			severity = "High"
		}
		result.Items = append(result.Items, Incident{fmt.Sprintf("evt-%d", i+1), severity, "Observed", event.Reason + " on " + event.Object, event.Message, event.Namespace, event.Object, event.LastSeen, []IncidentEvidence{{"Retained Kubernetes event", event.Message, event.LastSeen}}})
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
