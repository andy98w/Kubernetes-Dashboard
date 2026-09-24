package kubernetes

import (
	"context"
	"encoding/json"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"time"
)

type NodeFact struct {
	Name           string   `json:"name"`
	Zone           string   `json:"zone"`
	CPUCapacity    int64    `json:"cpuCapacity"`
	MemoryCapacity int64    `json:"memoryCapacity"`
	CPUMilli       *int64   `json:"cpuMilli"`
	MemoryBytes    *int64   `json:"memoryBytes"`
	Pressure       []string `json:"pressure"`
}
type VolumeFact struct {
	Namespace    string `json:"namespace"`
	Pod          string `json:"pod"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Claim        string `json:"claim"`
	Phase        string `json:"phase"`
	Capacity     string `json:"capacity"`
	StorageClass string `json:"storageClass"`
}
type PendingFact struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}
type TopologyFacts struct {
	Nodes   []NodeFact    `json:"nodes"`
	Volumes []VolumeFact  `json:"volumes"`
	Pending []PendingFact `json:"pending"`
}

func (c *Client) topologyFacts(ctx context.Context, pods []core.Pod, out *Topology) {
	out.Nodes = []NodeFact{}
	out.Volumes = []VolumeFact{}
	out.Pending = []PendingFact{}
	nodes, err := c.client.CoreV1().Nodes().List(ctx, meta.ListOptions{})
	if err != nil {
		out.Warnings = append(out.Warnings, "Node capacity and AZ discovery unavailable")
	} else {
		for _, n := range nodes.Items {
			f := NodeFact{Name: n.Name, Zone: n.Labels["topology.kubernetes.io/zone"], CPUCapacity: n.Status.Allocatable.Cpu().MilliValue(), MemoryCapacity: n.Status.Allocatable.Memory().Value(), Pressure: []string{}}
			for _, condition := range n.Status.Conditions {
				if condition.Status == core.ConditionTrue && (condition.Type == core.NodeMemoryPressure || condition.Type == core.NodeDiskPressure || condition.Type == core.NodePIDPressure) {
					f.Pressure = append(f.Pressure, string(condition.Type))
				}
			}
			out.Nodes = append(out.Nodes, f)
		}
	}
	claims, err := c.client.CoreV1().PersistentVolumeClaims("").List(ctx, meta.ListOptions{})
	byClaim := map[string]core.PersistentVolumeClaim{}
	if err != nil {
		out.Warnings = append(out.Warnings, "PVC metadata unavailable")
	} else {
		for _, claim := range claims.Items {
			byClaim[claim.Namespace+"/"+claim.Name] = claim
		}
	}
	for _, p := range pods {
		if p.Spec.NodeName == "" && p.Status.Phase != core.PodSucceeded && p.Status.Phase != core.PodFailed {
			f := PendingFact{Namespace: p.Namespace, Pod: p.Name, Reason: "Scheduling information unavailable"}
			for _, condition := range p.Status.Conditions {
				if condition.Type == core.PodScheduled && condition.Status == core.ConditionFalse {
					f.Reason = condition.Reason
					f.Message = condition.Message
				}
			}
			out.Pending = append(out.Pending, f)
		}
		mounted := map[string]bool{}
		for _, container := range append(append([]core.Container{}, p.Spec.Containers...), p.Spec.InitContainers...) {
			for _, m := range container.VolumeMounts {
				mounted[m.Name] = true
			}
			for _, m := range container.VolumeDevices {
				mounted[m.Name] = true
			}
		}
		for _, v := range p.Spec.Volumes {
			if !mounted[v.Name] {
				continue
			}
			f := VolumeFact{Namespace: p.Namespace, Pod: p.Name, Name: v.Name, Phase: "Unknown"}
			if v.PersistentVolumeClaim != nil {
				f.Kind = "PVC"
				f.Claim = v.PersistentVolumeClaim.ClaimName
				if claim, ok := byClaim[p.Namespace+"/"+f.Claim]; ok {
					f.Phase = string(claim.Status.Phase)
					if size, ok := claim.Status.Capacity[core.ResourceStorage]; ok {
						f.Capacity = size.String()
					}
					if claim.Spec.StorageClassName != nil {
						f.StorageClass = *claim.Spec.StorageClassName
					}
				}
			} else if v.EmptyDir != nil {
				f.Kind = "emptyDir"
				f.Phase = "Pod lifetime"
			} else {
				continue
			}
			out.Volumes = append(out.Volumes, f)
		}
	}
	// Metrics Server is optional. Missing, forbidden and stale samples stay null.
	if rc, ok := c.client.CoreV1().RESTClient().(*rest.RESTClient); ok && rc != nil {
		metricCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		raw, e := rc.Get().AbsPath("/apis/metrics.k8s.io/v1beta1/nodes").DoRaw(metricCtx)
		var payload struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
				Timestamp time.Time                    `json:"timestamp"`
				Usage     map[string]resource.Quantity `json:"usage"`
			} `json:"items"`
		}
		if e == nil {
			e = json.Unmarshal(raw, &payload)
		}
		if e != nil {
			out.Warnings = append(out.Warnings, "Node usage metrics unavailable")
		} else {
			for _, m := range payload.Items {
				age := time.Since(m.Timestamp)
				if age < 0 || age > 2*time.Minute {
					continue
				}
				for i := range out.Nodes {
					if out.Nodes[i].Name == m.Metadata.Name {
						if q, ok := m.Usage["cpu"]; ok {
							v := q.MilliValue()
							out.Nodes[i].CPUMilli = &v
						}
						if q, ok := m.Usage["memory"]; ok {
							v := q.Value()
							out.Nodes[i].MemoryBytes = &v
						}
					}
				}
			}
		}
	}
}
