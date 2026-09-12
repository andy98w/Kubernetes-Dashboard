package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) previousTemplate(ctx context.Context, deployment *appsv1.Deployment) (*corev1.PodTemplateSpec, string, error) {
	current, err := strconv.Atoi(deployment.Annotations["deployment.kubernetes.io/revision"])
	if err != nil || current < 2 || deployment.UID == "" {
		return nil, "", fmt.Errorf("no previous owned revision is available")
	}
	sets, err := c.client.AppsV1().ReplicaSets(deployment.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, "", err
	}
	var best *appsv1.ReplicaSet
	revision := 0
	for i := range sets.Items {
		rs := &sets.Items[i]
		owner := metav1.GetControllerOf(rs)
		if owner == nil || owner.UID != deployment.UID || owner.Kind != "Deployment" {
			continue
		}
		n, e := strconv.Atoi(rs.Annotations["deployment.kubernetes.io/revision"])
		if e == nil && n > revision && n < current {
			best, revision = rs, n
		}
	}
	if best == nil {
		return nil, "", fmt.Errorf("previous ReplicaSet was not found; history may have been pruned")
	}
	template := best.Spec.Template.DeepCopy()
	delete(template.Labels, "pod-template-hash")
	return template, strconv.Itoa(revision), nil
}

// Report every changed field path without disclosing environment variable values.
func templateChanges(before, after corev1.PodTemplateSpec) []string {
	var left, right any
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	_ = json.Unmarshal(b, &left)
	_ = json.Unmarshal(a, &right)
	changes := []string{}
	var visit func(string, any, any)
	visit = func(path string, l, r any) {
		if reflect.DeepEqual(l, r) {
			return
		}
		lm, lok := l.(map[string]any)
		rm, rok := r.(map[string]any)
		if lok && rok {
			keys := map[string]bool{}
			for k := range lm {
				keys[k] = true
			}
			for k := range rm {
				keys[k] = true
			}
			for k := range keys {
				visit(path+"/"+k, lm[k], rm[k])
			}
			return
		}
		la, lok := l.([]any)
		ra, rok := r.([]any)
		if lok && rok && len(la) == len(ra) {
			for i := range la {
				visit(fmt.Sprintf("%s/%d", path, i), la[i], ra[i])
			}
			return
		}
		changes = append(changes, path)
	}
	visit("/spec/template", left, right)
	sort.Strings(changes)
	return changes
}

type Recovery struct {
	Status             string `json:"status"`
	Detail             string `json:"detail"`
	Generation         int64  `json:"generation"`
	ObservedGeneration int64  `json:"observedGeneration"`
}

func deploymentRecovery(d *appsv1.Deployment) Recovery {
	r := Recovery{Status: "Progressing", Generation: d.Generation, ObservedGeneration: d.Status.ObservedGeneration}
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	r.Detail = fmt.Sprintf("%d/%d updated, %d available; controller observed generation %d of %d.", d.Status.UpdatedReplicas, desired, d.Status.AvailableReplicas, d.Status.ObservedGeneration, d.Generation)
	if d.Spec.Paused {
		r.Status = "Paused"
		return r
	}
	if d.Status.ObservedGeneration < d.Generation {
		return r
	}
	for _, condition := range d.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionFalse && condition.Reason == "ProgressDeadlineExceeded" {
			r.Status = "Stalled"
			return r
		}
	}
	if d.Status.UpdatedReplicas == desired && d.Status.AvailableReplicas == desired && d.Status.Replicas == desired && d.Status.UnavailableReplicas == 0 {
		r.Status = "Recovered"
	}
	return r
}
