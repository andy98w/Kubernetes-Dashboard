package kubernetes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	OperationRestart  = "restart"
	OperationScale    = "scale"
	OperationRollback = "rollback"
)

type OperationRequest struct {
	rollbackTemplate        *corev1.PodTemplateSpec
	Action                  string `json:"action"`
	Namespace               string `json:"namespace"`
	Kind                    string `json:"kind"`
	Name                    string `json:"name"`
	Replicas                *int32 `json:"replicas,omitempty"`
	Reason                  string `json:"reason"`
	PlanID                  string `json:"planId,omitempty"`
	ExpectedResourceVersion string `json:"expectedResourceVersion,omitempty"`
}

type OperationPlan struct {
	RollbackRevision string    `json:"rollbackRevision,omitempty"`
	TemplateDiff     []string  `json:"templateDiff,omitempty"`
	ID               string    `json:"id"`
	Action           string    `json:"action"`
	Target           string    `json:"target"`
	Reason           string    `json:"reason"`
	CurrentReplicas  int32     `json:"currentReplicas"`
	DesiredReplicas  int32     `json:"desiredReplicas"`
	Impact           string    `json:"impact"`
	ResourceVersion  string    `json:"resourceVersion"`
	Authorized       bool      `json:"authorized"`
	DryRun           bool      `json:"dryRun"`
	Simulated        bool      `json:"simulated"`
	CreatedAt        time.Time `json:"createdAt"`
}

type OperationRecord struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Actor     string    `json:"actor"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"`
	Before    string    `json:"before"`
	After     string    `json:"after"`
	Simulated bool      `json:"simulated"`
	CreatedAt time.Time `json:"createdAt"`
}

type OperationError struct {
	Code    string
	Message string
}

func (e *OperationError) Error() string { return e.Message }

type operationPolicy struct {
	enabled                  bool
	namespaces               map[string]bool
	minReplicas, maxReplicas int32
}
type operationAudit struct {
	mu      sync.RWMutex
	records []OperationRecord
}

type operationPlanEntry struct {
	request OperationRequest
	plan    OperationPlan
}

type operationPlans struct {
	mu      sync.Mutex
	entries map[string]operationPlanEntry
}

func (c *Client) PlanOperation(ctx context.Context, request OperationRequest) (OperationPlan, error) {
	if err := c.validateOperation(request); err != nil {
		return OperationPlan{}, err
	}
	deployment, err := c.client.AppsV1().Deployments(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return OperationPlan{}, &OperationError{"target_unavailable", "deployment could not be read"}
	}
	if deployment.Spec.Paused && request.Action != OperationScale {
		return OperationPlan{}, &OperationError{"deployment_paused", "resume the Deployment through its source configuration before restarting or rolling back"}
	}
	revision := ""
	changes := []string{}
	if request.Action == OperationRollback {
		template, rev, err := c.previousTemplate(ctx, deployment)
		if err != nil {
			return OperationPlan{}, &OperationError{"rollback_unavailable", err.Error()}
		}
		request.rollbackTemplate, revision = template, rev
		changes = templateChanges(deployment.Spec.Template, *template)
	}
	current := int32(1)
	if deployment.Spec.Replicas != nil {
		current = *deployment.Spec.Replicas
	}
	desired := current
	if request.Action == OperationScale {
		desired = *request.Replicas
	}
	if err := c.applyOperation(ctx, request, deployment.ResourceVersion, true); err != nil {
		return OperationPlan{}, &OperationError{"dry_run_rejected", "Kubernetes rejected the dry run: " + err.Error()}
	}
	impact := "Recreates each pod through the Deployment's rollout strategy; service traffic remains controller-managed."
	if request.Action == OperationScale {
		impact = fmt.Sprintf("Changes desired replicas from %d to %d within the configured %d–%d guardrail. A HorizontalPodAutoscaler, if present, may subsequently reconcile this value.", current, desired, c.operations.minReplicas, c.operations.maxReplicas)
	}
	if request.Action == OperationRollback {
		impact = "Restores the exact pod template from owned ReplicaSet revision " + revision + ". Previous does not mean healthy. Review all template changes; GitOps may restore the source configuration."
	}
	plan := OperationPlan{ID: newOperationID(), Action: request.Action, Target: target(request), Reason: request.Reason, CurrentReplicas: current, DesiredReplicas: desired, Impact: impact, ResourceVersion: deployment.ResourceVersion, Authorized: true, DryRun: true, CreatedAt: time.Now().UTC(), RollbackRevision: revision, TemplateDiff: changes}
	if c.durable != nil {
		if err := c.saveDurablePlan(ctx, request, plan, deployment); err != nil {
			return OperationPlan{}, err
		}
	} else {
		c.rememberOperationPlan(request, plan)
	}
	return plan, nil
}

func (c *Client) ExecuteOperation(ctx context.Context, request OperationRequest, actor string) (OperationRecord, error) {
	if c.durable != nil {
		return c.approveDurable(ctx, request, actor)
	}
	if request.PlanID == "" || request.ExpectedResourceVersion == "" {
		return OperationRecord{}, &OperationError{"plan_required", "review the operation again before executing it"}
	}
	if err := c.validateOperation(request); err != nil {
		return OperationRecord{}, err
	}
	if err := c.consumeOperationPlan(&request); err != nil {
		return OperationRecord{}, err
	}
	deployment, err := c.client.AppsV1().Deployments(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return OperationRecord{}, &OperationError{"target_unavailable", "deployment could not be read"}
	}
	if deployment.ResourceVersion != request.ExpectedResourceVersion {
		return OperationRecord{}, &OperationError{"stale_plan", "the deployment changed after review; review the operation again"}
	}
	current := int32(1)
	if deployment.Spec.Replicas != nil {
		current = *deployment.Spec.Replicas
	}
	if err := c.applyOperation(ctx, request, deployment.ResourceVersion, true); err != nil {
		return OperationRecord{}, &OperationError{"dry_run_rejected", "Kubernetes rejected the final dry run: " + err.Error()}
	}
	if err := c.applyOperation(ctx, request, deployment.ResourceVersion, false); err != nil {
		return OperationRecord{}, &OperationError{"execution_failed", "Kubernetes rejected the operation: " + err.Error()}
	}
	after := fmt.Sprintf("%d replicas", current)
	if request.Action == OperationScale {
		after = fmt.Sprintf("%d replicas", *request.Replicas)
	}
	if request.Action == OperationRollback {
		after = "Reviewed previous pod template restored; recovery not yet verified"
	}
	record := OperationRecord{request.PlanID, request.Action, target(request), actor, request.Reason, "Accepted", fmt.Sprintf("%d replicas", current), after, false, time.Now().UTC()}
	c.appendOperation(record)
	slog.Info("guarded Kubernetes operation accepted", "operation_id", record.ID, "actor", actor, "action", request.Action, "target", record.Target, "reason", request.Reason, "before", record.Before, "after", record.After)
	return record, nil
}

func (c *Client) RecentOperations(ctx context.Context) ([]OperationRecord, error) {
	if c.durable != nil {
		return c.durableRecords(ctx)
	}
	c.audit.mu.RLock()
	defer c.audit.mu.RUnlock()
	return append([]OperationRecord(nil), c.audit.records...), nil
}

func (c *Client) validateOperation(request OperationRequest) error {
	if !c.operations.enabled {
		return &OperationError{"operations_disabled", "guarded operations are disabled for this deployment"}
	}
	if !c.operations.namespaces[request.Namespace] {
		return &OperationError{"namespace_denied", "operations are not enabled for this namespace"}
	}
	if !strings.EqualFold(request.Kind, "Deployment") {
		return &OperationError{"kind_denied", "only Deployments support guarded operations"}
	}
	if request.Name == "" || len(request.Name) > 253 {
		return &OperationError{"invalid_target", "deployment name is required"}
	}
	reason := strings.TrimSpace(request.Reason)
	if len(reason) < 8 || len(reason) > 200 {
		return &OperationError{"invalid_reason", "reason must be between 8 and 200 characters"}
	}
	if request.Action != OperationRestart && request.Action != OperationScale && request.Action != OperationRollback {
		return &OperationError{"action_denied", "operation is not supported"}
	}
	if request.Action == OperationScale {
		if request.Replicas == nil {
			return &OperationError{"invalid_replicas", "replica count is required"}
		}
		if *request.Replicas < c.operations.minReplicas || *request.Replicas > c.operations.maxReplicas {
			return &OperationError{"invalid_replicas", fmt.Sprintf("replicas must be between %d and %d", c.operations.minReplicas, c.operations.maxReplicas)}
		}
	}
	return nil
}

func (c *Client) applyOperation(ctx context.Context, request OperationRequest, resourceVersion string, dryRun bool) error {
	dryRunValues := []string(nil)
	if dryRun {
		dryRunValues = []string{metav1.DryRunAll}
	}
	if request.Action == OperationScale {
		scale, err := c.client.AppsV1().Deployments(request.Namespace).GetScale(ctx, request.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if resourceVersion != "" && scale.ResourceVersion != "" && scale.ResourceVersion != resourceVersion {
			return fmt.Errorf("resource version changed")
		}
		scale.Spec.Replicas = *request.Replicas
		_, err = c.client.AppsV1().Deployments(request.Namespace).UpdateScale(ctx, request.Name, scale, metav1.UpdateOptions{DryRun: dryRunValues, FieldManager: "kubevista"})
		return err
	}
	if request.Action == OperationRollback {
		if request.rollbackTemplate == nil {
			return fmt.Errorf("reviewed rollback template is missing")
		}
		payload, err := json.Marshal([]map[string]any{
			{"op": "test", "path": "/metadata/resourceVersion", "value": resourceVersion},
			{"op": "replace", "path": "/spec/template", "value": request.rollbackTemplate},
		})
		if err != nil {
			return err
		}
		_, err = c.client.AppsV1().Deployments(request.Namespace).Patch(ctx, request.Name, types.JSONPatchType, payload, metav1.PatchOptions{DryRun: dryRunValues, FieldManager: "kubevista"})
		return err
	}
	payload, _ := json.Marshal(map[string]any{"metadata": map[string]any{"resourceVersion": resourceVersion}, "spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]string{"kubevista.io/restart-request": request.PlanID, "kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339Nano)}}}}})
	_, err := c.client.AppsV1().Deployments(request.Namespace).Patch(ctx, request.Name, types.MergePatchType, payload, metav1.PatchOptions{DryRun: dryRunValues, FieldManager: "kubevista"})
	return err
}

func (c *Client) appendOperation(record OperationRecord) {
	c.audit.mu.Lock()
	defer c.audit.mu.Unlock()
	c.audit.records = append([]OperationRecord{record}, c.audit.records...)
	if len(c.audit.records) > 50 {
		c.audit.records = c.audit.records[:50]
	}
}

func (c *Client) rememberOperationPlan(request OperationRequest, plan OperationPlan) {
	c.plans.mu.Lock()
	defer c.plans.mu.Unlock()
	cutoff := time.Now().UTC().Add(-5 * time.Minute)
	oldestID := ""
	oldestTime := time.Now().UTC()
	for id, entry := range c.plans.entries {
		if entry.plan.CreatedAt.Before(cutoff) {
			delete(c.plans.entries, id)
			continue
		}
		if entry.plan.CreatedAt.Before(oldestTime) {
			oldestID, oldestTime = id, entry.plan.CreatedAt
		}
	}
	if len(c.plans.entries) >= 100 && oldestID != "" {
		delete(c.plans.entries, oldestID)
	}
	request.PlanID = ""
	request.ExpectedResourceVersion = ""
	c.plans.entries[plan.ID] = operationPlanEntry{request: request, plan: plan}
}

func (c *Client) consumeOperationPlan(supplied *OperationRequest) error {
	request := *supplied
	c.plans.mu.Lock()
	defer c.plans.mu.Unlock()
	entry, ok := c.plans.entries[request.PlanID]
	if !ok || time.Since(entry.plan.CreatedAt) > 5*time.Minute {
		delete(c.plans.entries, request.PlanID)
		return &OperationError{"plan_invalid", "the operation plan is missing or expired; review the operation again"}
	}
	delete(c.plans.entries, request.PlanID)
	expected := entry.request
	suppliedResourceVersion := request.ExpectedResourceVersion
	request.PlanID = ""
	request.ExpectedResourceVersion = ""
	if suppliedResourceVersion == "" || suppliedResourceVersion != entry.plan.ResourceVersion {
		return &OperationError{"plan_invalid", "the operation plan is invalid; review the operation again"}
	}
	if request.Action != expected.Action || request.Namespace != expected.Namespace || !strings.EqualFold(request.Kind, expected.Kind) || request.Name != expected.Name || strings.TrimSpace(request.Reason) != strings.TrimSpace(expected.Reason) || !sameReplicas(request.Replicas, expected.Replicas) {
		return &OperationError{"plan_mismatch", "the operation changed after review; review it again before executing"}
	}
	supplied.rollbackTemplate = expected.rollbackTemplate
	return nil
}

func sameReplicas(left, right *int32) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func target(request OperationRequest) string {
	return request.Namespace + "/Deployment/" + request.Name
}
func newOperationID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("op-%d", time.Now().UnixNano())
	}
	return "op-" + hex.EncodeToString(b)
}
