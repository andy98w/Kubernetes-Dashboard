package kubernetes

import (
	"context"
	networking "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"time"
)

type RouteEdge struct {
	Namespace string `json:"namespace"`
	Ingress   string `json:"ingress"`
	Host      string `json:"host"`
	Path      string `json:"path"`
	Service   string `json:"service"`
	Port      string `json:"port"`
}
type EndpointEdge struct {
	Namespace string   `json:"namespace"`
	Service   string   `json:"service"`
	Pod       string   `json:"pod"`
	PodUID    string   `json:"podUID"`
	Addresses []string `json:"addresses"`
	Ready     *bool    `json:"ready"`
}
type IdentityGrant struct {
	Scope   string            `json:"scope"`
	Binding string            `json:"binding"`
	Rules   []rbac.PolicyRule `json:"rules"`
}
type PodIdentity struct {
	Namespace      string          `json:"namespace"`
	Pod            string          `json:"pod"`
	ServiceAccount string          `json:"serviceAccount"`
	Grants         []IdentityGrant `json:"grants"`
	Policies       []string        `json:"policies"`
}
type PolicyDetail struct {
	Namespace string                       `json:"namespace"`
	Name      string                       `json:"name"`
	Spec      networking.NetworkPolicySpec `json:"spec"`
}
type Topology struct {
	TopologyFacts
	Services      []Service         `json:"services"`
	Routes        []RouteEdge       `json:"routes"`
	Endpoints     []EndpointEdge    `json:"endpoints"`
	Identities    []PodIdentity     `json:"identities"`
	Policies      []PolicyDetail    `json:"policies"`
	ExternalNames map[string]string `json:"externalNames"`
	Warnings      []string          `json:"warnings"`
	ObservedAt    time.Time         `json:"observedAt"`
}

// Topology reports configuration and discovery evidence, not access decisions
// or measured traffic. No token, Secret, or pod environment is read.
func (c *Client) Topology(ctx context.Context) (Topology, error) {
	out := Topology{Services: []Service{}, Routes: []RouteEdge{}, Endpoints: []EndpointEdge{}, Identities: []PodIdentity{}, Policies: []PolicyDetail{}, ExternalNames: map[string]string{}, Warnings: []string{}, ObservedAt: time.Now().UTC()}
	ing, err := c.client.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		out.Warnings = append(out.Warnings, "Ingress discovery unavailable")
	} else {
		for _, i := range ing.Items {
			add := func(host, path string, b networking.IngressBackend) {
				if b.Service != nil {
					port := b.Service.Port.Name
					if port == "" {
						port = strconv.Itoa(int(b.Service.Port.Number))
					}
					out.Routes = append(out.Routes, RouteEdge{i.Namespace, i.Name, host, path, b.Service.Name, port})
				}
			}
			if i.Spec.DefaultBackend != nil {
				add("", "(default)", *i.Spec.DefaultBackend)
			}
			for _, rule := range i.Spec.Rules {
				if rule.HTTP != nil {
					for _, p := range rule.HTTP.Paths {
						add(rule.Host, p.Path, p.Backend)
					}
				}
			}
		}
	}
	slices, err := c.client.DiscoveryV1().EndpointSlices("").List(ctx, metav1.ListOptions{})
	if err != nil {
		out.Warnings = append(out.Warnings, "EndpointSlice discovery unavailable")
	} else {
		for _, s := range slices.Items {
			service := s.Labels["kubernetes.io/service-name"]
			if service == "" {
				continue
			}
			for _, e := range s.Endpoints {
				pod, uid := "", ""
				if e.TargetRef != nil && e.TargetRef.Kind == "Pod" && (e.TargetRef.Namespace == "" || e.TargetRef.Namespace == s.Namespace) {
					pod = e.TargetRef.Name
					uid = string(e.TargetRef.UID)
				}
				out.Endpoints = append(out.Endpoints, EndpointEdge{s.Namespace, service, pod, uid, e.Addresses, e.Conditions.Ready})
			}
		}
	}
	services, err := c.client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		out.Warnings = append(out.Warnings, "Service and ExternalName discovery unavailable")
	} else {
		for _, s := range services.Items {
			ports := []string{}
			for _, p := range s.Spec.Ports {
				ports = append(ports, strconv.Itoa(int(p.Port))+"/"+string(p.Protocol))
			}
			out.Services = append(out.Services, Service{Namespace: s.Namespace, Name: s.Name, Type: string(s.Spec.Type), ClusterIP: s.Spec.ClusterIP, Ports: ports})
			if s.Spec.ExternalName != "" {
				out.ExternalNames[s.Namespace+"/"+s.Name] = s.Spec.ExternalName
			}
		}
	}
	policies, err := c.client.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err != nil {
		out.Warnings = append(out.Warnings, "NetworkPolicy discovery unavailable")
	} else {
		for _, p := range policies.Items {
			out.Policies = append(out.Policies, PolicyDetail{p.Namespace, p.Name, p.Spec})
		}
	}
	roles, re := c.client.RbacV1().Roles("").List(ctx, metav1.ListOptions{})
	clusters, ce := c.client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	bindings, be := c.client.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	clusterBindings, cbe := c.client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if re != nil || ce != nil || be != nil || cbe != nil {
		out.Warnings = append(out.Warnings, "RBAC discovery incomplete; displayed grants are not an effective authorization decision")
	}
	rules := map[string][]rbac.PolicyRule{}
	if re == nil {
		for _, r := range roles.Items {
			rules[r.Namespace+"/"+r.Name] = r.Rules
		}
	}
	if ce == nil {
		for _, r := range clusters.Items {
			rules["/"+r.Name] = r.Rules
		}
	}
	pods, err := c.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		out.Warnings = append(out.Warnings, "Pod identity discovery unavailable")
	} else {
		for _, p := range pods.Items {
			id := PodIdentity{p.Namespace, p.Name, p.Spec.ServiceAccountName, []IdentityGrant{}, []string{}}
			for _, policy := range out.Policies {
				if policy.Namespace != p.Namespace {
					continue
				}
				selector, e := metav1.LabelSelectorAsSelector(&policy.Spec.PodSelector)
				if e == nil && selector.Matches(labels.Set(p.Labels)) {
					id.Policies = append(id.Policies, policy.Name)
				}
			}
			add := func(scope, name string, subjects []rbac.Subject, ref rbac.RoleRef) {
				if ref.APIGroup != "rbac.authorization.k8s.io" {
					return
				}
				matched := false
				for _, s := range subjects {
					if s.Kind == "ServiceAccount" && s.Name == id.ServiceAccount && s.Namespace == p.Namespace {
						matched = true
					}
					if s.Kind == "User" && s.Name == "system:serviceaccount:"+p.Namespace+":"+id.ServiceAccount {
						matched = true
					}
					if s.Kind == "Group" && (s.Name == "system:authenticated" || s.Name == "system:serviceaccounts" || s.Name == "system:serviceaccounts:"+p.Namespace) {
						matched = true
					}
				}
				if !matched {
					return
				}
				key := "/" + ref.Name
				if ref.Kind == "Role" {
					key = scope + "/" + ref.Name
				}
				if rr, ok := rules[key]; ok {
					id.Grants = append(id.Grants, IdentityGrant{scope, name, rr})
				}
			}
			if be == nil {
				for _, b := range bindings.Items {
					add(b.Namespace, b.Namespace+"/"+b.Name, b.Subjects, b.RoleRef)
				}
			}
			if cbe == nil {
				for _, b := range clusterBindings.Items {
					add("*", b.Name, b.Subjects, b.RoleRef)
				}
			}
			out.Identities = append(out.Identities, id)
		}
	}
	if pods != nil {
		c.topologyFacts(ctx, pods.Items, &out)
	} else {
		c.topologyFacts(ctx, nil, &out)
	}
	return out, nil
}
