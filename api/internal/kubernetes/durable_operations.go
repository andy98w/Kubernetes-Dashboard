package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const operationLabel = "kubevista.io/durable-operation"
const operationMarker = "kubevista.io/operation-id"
const claimDuration = 15 * time.Second

type durableOperation struct {
	Request    OperationRequest       `json:"request"`
	Plan       OperationPlan          `json:"plan"`
	TargetUID  types.UID              `json:"targetUid"`
	Template   corev1.PodTemplateSpec `json:"template"`
	Replicas   int32                  `json:"replicas"`
	State      string                 `json:"state"`
	Actor      string                 `json:"actor,omitempty"`
	Owner      string                 `json:"owner,omitempty"`
	Epoch      int64                  `json:"epoch"`
	ClaimUntil time.Time              `json:"claimUntil"`
	ApprovedAt time.Time              `json:"approvedAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	Detail     string                 `json:"detail,omitempty"`
	History    []operationTransition  `json:"history"`
}
type operationTransition struct {
	State  string    `json:"state"`
	At     time.Time `json:"at"`
	Epoch  int64     `json:"epoch"`
	Detail string    `json:"detail"`
}
type durableController struct {
	namespace, identity string
	activeActive        bool
	// Tests inject a process interruption after the target write but before receipt persistence.
	afterApply func() error
}

func (c *Client) enableDurable(namespace, identity string, activeActive bool) {
	identity = identity + "-" + newOperationID() // process incarnation, even when POD_NAME is reused.
	c.durable = &durableController{namespace: namespace, identity: identity, activeActive: activeActive}
}
func terminal(state string) bool {
	return state == "Succeeded" || state == "Failed" || state == "Expired" || state == "NeedsReview"
}
func transition(op *durableOperation, state, detail string) {
	op.State, op.Detail, op.UpdatedAt = state, detail, time.Now().UTC()
	op.History = append(op.History, operationTransition{state, op.UpdatedAt, op.Epoch, detail})
}
func operationData(op durableOperation) (map[string]string, error) {
	data, err := json.Marshal(op)
	if err != nil {
		return nil, err
	}
	if len(data) > 256*1024 {
		return nil, fmt.Errorf("operation record exceeds 256 KiB")
	}
	return map[string]string{"operation.json": string(data)}, nil
}
func decodeDurable(cm *corev1.ConfigMap) (durableOperation, error) {
	var op durableOperation
	if cm.Labels[operationLabel] != "true" {
		return op, fmt.Errorf("not an operation record")
	}
	err := json.Unmarshal([]byte(cm.Data["operation.json"]), &op)
	if err == nil && (op.Plan.ID != cm.Name || op.TargetUID == "" || op.Plan.ResourceVersion == "") {
		err = fmt.Errorf("invalid operation record")
	}
	return op, err
}
func (c *Client) saveDurablePlan(ctx context.Context, r OperationRequest, p OperationPlan, d *appsv1.Deployment) error {
	if d.UID == "" || d.ResourceVersion == "" {
		return fmt.Errorf("target identity is required")
	}
	template := d.Spec.Template.DeepCopy()
	if r.Action == OperationRollback {
		template = r.rollbackTemplate.DeepCopy()
	}
	if r.Action == OperationRestart {
		if template.Annotations == nil {
			template.Annotations = map[string]string{}
		}
		template.Annotations["kubevista.io/restart-request"] = p.ID
		template.Annotations["kubectl.kubernetes.io/restartedAt"] = p.CreatedAt.Format(time.RFC3339Nano)
	}
	r.PlanID, r.ExpectedResourceVersion = p.ID, p.ResourceVersion
	op := durableOperation{Request: r, Plan: p, TargetUID: d.UID, Template: *template, Replicas: p.DesiredReplicas}
	transition(&op, "Planned", "Awaiting explicit approval; expires five minutes after creation.")
	data, err := operationData(op)
	if err != nil {
		return err
	}
	_, err = c.client.CoreV1().ConfigMaps(c.durable.namespace).Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: p.ID, Labels: map[string]string{operationLabel: "true"}}, Data: data}, metav1.CreateOptions{})
	return err
}
func (c *Client) readDurable(ctx context.Context, id string) (*corev1.ConfigMap, durableOperation, error) {
	cm, err := c.client.CoreV1().ConfigMaps(c.durable.namespace).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, durableOperation{}, err
	}
	op, err := decodeDurable(cm)
	return cm, op, err
}
func (c *Client) updateDurable(ctx context.Context, cm *corev1.ConfigMap, op durableOperation) error {
	data, err := operationData(op)
	if err != nil {
		return err
	}
	copy := cm.DeepCopy()
	copy.Data = data
	_, err = c.client.CoreV1().ConfigMaps(c.durable.namespace).Update(ctx, copy, metav1.UpdateOptions{})
	return err // resourceVersion is a compare-and-swap; never rebase an old status write.
}
func recordOf(op durableOperation) OperationRecord {
	return OperationRecord{ID: op.Plan.ID, Action: op.Request.Action, Target: op.Plan.Target, Actor: op.Actor, Reason: op.Plan.Reason, Status: op.State, Before: fmt.Sprintf("%d replicas", op.Plan.CurrentReplicas), After: op.Detail, CreatedAt: op.UpdatedAt}
}
func (c *Client) approveDurable(ctx context.Context, r OperationRequest, actor string) (OperationRecord, error) {
	if err := c.validateOperation(r); err != nil {
		return OperationRecord{}, err
	}
	if r.PlanID == "" || r.ExpectedResourceVersion == "" || actor == "" {
		return OperationRecord{}, &OperationError{"plan_required", "review and authenticated approval are required"}
	}
	cm, op, err := c.readDurable(ctx, r.PlanID)
	if err != nil {
		return OperationRecord{}, &OperationError{"plan_invalid", "operation plan unavailable"}
	}
	expected := op.Request
	if r.Action != expected.Action || r.Namespace != expected.Namespace || !strings.EqualFold(r.Kind, expected.Kind) || r.Name != expected.Name || strings.TrimSpace(r.Reason) != strings.TrimSpace(expected.Reason) || !sameReplicas(r.Replicas, expected.Replicas) || r.ExpectedResourceVersion != op.Plan.ResourceVersion {
		return OperationRecord{}, &OperationError{"plan_mismatch", "request differs from reviewed plan"}
	}
	if op.State != "Planned" {
		if op.Actor == actor && op.Actor != "" {
			return recordOf(op), nil
		} // idempotent replay, never another execution.
		return OperationRecord{}, &OperationError{"plan_invalid", "plan is no longer awaiting approval"}
	}
	if time.Since(op.Plan.CreatedAt) > 5*time.Minute {
		transition(&op, "Expired", "Approval window expired.")
		_ = c.updateDurable(ctx, cm, op)
		return OperationRecord{}, &OperationError{"plan_invalid", "plan expired; review again"}
	}
	op.Actor, op.ApprovedAt = actor, time.Now().UTC()
	transition(&op, "Approved", "Approved; queued for reconciliation, not yet applied.")
	if err = c.updateDurable(ctx, cm, op); err != nil {
		return OperationRecord{}, &OperationError{"stale_plan", "approval raced with another update; retry the same plan"}
	}
	return recordOf(op), nil
}
func (c *Client) durableRecords(ctx context.Context) ([]OperationRecord, error) {
	list, err := c.client.CoreV1().ConfigMaps(c.durable.namespace).List(ctx, metav1.ListOptions{LabelSelector: operationLabel + "=true"})
	if err != nil {
		return nil, err
	}
	records := []OperationRecord{}
	for i := range list.Items {
		op, err := decodeDurable(&list.Items[i])
		if err == nil {
			records = append(records, recordOf(op))
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt.After(records[j].CreatedAt) })
	if len(records) > 50 {
		records = records[:50]
	}
	return records, nil
}

// ReconcileOperation is level-based: no watcher event is trusted as current state.
func (c *Client) ReconcileOperation(ctx context.Context, id string) error {
	cm, op, err := c.readDurable(ctx, id)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if terminal(op.State) {
		return nil
	}
	if op.State == "Planned" {
		if time.Since(op.Plan.CreatedAt) > 5*time.Minute {
			transition(&op, "Expired", "Approval window expired.")
			return c.updateDurable(ctx, cm, op)
		}
		return nil
	}
	if op.State != "Approved" && op.State != "Executing" && op.State != "Verifying" {
		return fmt.Errorf("unknown operation state")
	}
	if op.Owner != "" && op.Owner != c.durable.identity && time.Now().Before(op.ClaimUntil) {
		return nil
	}
	// Claim epochs fence receipt writers. Deployment UID/resourceVersion fences effects.
	if op.Owner != c.durable.identity || time.Now().After(op.ClaimUntil) {
		op.Epoch++
	}
	op.Owner = c.durable.identity
	op.ClaimUntil = time.Now().UTC().Add(claimDuration)
	if op.State == "Approved" {
		transition(&op, "Executing", "Claimed by controller; Kubernetes outcome not yet known.")
	}
	if err = c.updateDurable(ctx, cm, op); err != nil {
		return err
	}
	cm, claimed, err := c.readDurable(ctx, id)
	if err != nil {
		return err
	}
	if claimed.Owner != op.Owner || claimed.Epoch != op.Epoch {
		return fmt.Errorf("claim lost")
	}
	op = claimed
	finish := func(state, detail string) error { transition(&op, state, detail); return c.updateDurable(ctx, cm, op) }
	if err = c.validateOperation(op.Request); err != nil {
		return finish("NeedsReview", "Policy changed; no additional writes will be attempted.")
	}
	d, err := c.client.AppsV1().Deployments(op.Request.Namespace).Get(ctx, op.Request.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return finish("NeedsReview", "Target was deleted.")
	}
	if err != nil {
		return err
	}
	if d.UID != op.TargetUID {
		return finish("NeedsReview", "Target UID changed; refusing to modify replacement.")
	}
	if d.Annotations[operationMarker] == op.Plan.ID {
		replicas := int32(1)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		if replicas != op.Replicas || !reflect.DeepEqual(d.Spec.Template, op.Template) {
			return finish("NeedsReview", "Target intent drifted after application; not automatically repaired.")
		}
		recovery := deploymentRecovery(d)
		if recovery.Status == "Recovered" {
			return finish("Succeeded", recovery.Detail)
		}
		if recovery.Status == "Stalled" || time.Since(op.ApprovedAt) > 10*time.Minute {
			return finish("NeedsReview", "Applied but recovery is not established: "+recovery.Detail)
		}
		if op.State != "Verifying" {
			return finish("Verifying", recovery.Detail)
		}
		return nil
	}
	if op.State == "Verifying" {
		return finish("NeedsReview", "Operation marker changed; a newer change may have superseded this operation.")
	}
	if d.ResourceVersion != op.Plan.ResourceVersion {
		return finish("NeedsReview", "Target changed after review; outcome may be superseded. No rebase or replay.")
	}
	if time.Since(op.ApprovedAt) > 10*time.Minute {
		return finish("NeedsReview", "Execution window elapsed; outcome requires review.")
	}
	if d.Spec.Paused && op.Request.Action != OperationScale {
		return finish("NeedsReview", "Deployment is paused.")
	}
	if _, err = c.patchDurable(ctx, op, true); err != nil {
		if apierrors.IsForbidden(err) || apierrors.IsInvalid(err) {
			return finish("Failed", "Final Kubernetes dry run rejected.")
		}
		return err
	}
	// Re-read the claim immediately before dispatch. It cannot atomically fence an
	// in-flight request; the immutable target resourceVersion provides that boundary.
	_, fresh, err := c.readDurable(ctx, id)
	if err != nil {
		return err
	}
	if fresh.Owner != op.Owner || fresh.Epoch != op.Epoch || time.Now().After(fresh.ClaimUntil) {
		return fmt.Errorf("claim expired before dispatch")
	}
	if _, err = c.patchDurable(ctx, op, false); err != nil {
		return err
	} // ambiguous errors are reconciled, never blindly retried.
	if c.durable.afterApply != nil {
		if err = c.durable.afterApply(); err != nil {
			return err
		}
	}
	return finish("Verifying", "Kubernetes accepted the reviewed intent; rollout recovery pending.")
}

func (c *Client) patchDurable(ctx context.Context, op durableOperation, dry bool) (*appsv1.Deployment, error) {
	annotations := map[string]string{operationMarker: op.Plan.ID}
	// JSON patch adds one annotation without replacing unrelated annotations.
	d, err := c.client.AppsV1().Deployments(op.Request.Namespace).Get(ctx, op.Request.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	for k, v := range d.Annotations {
		if k != operationMarker {
			annotations[k] = v
		}
	}
	patch := []map[string]any{
		{"op": "test", "path": "/metadata/uid", "value": op.TargetUID},
		{"op": "test", "path": "/metadata/resourceVersion", "value": op.Plan.ResourceVersion},
		{"op": "add", "path": "/metadata/annotations", "value": annotations},
	}
	if op.Request.Action == OperationScale {
		patch = append(patch, map[string]any{"op": "add", "path": "/spec/replicas", "value": op.Replicas})
	} else {
		patch = append(patch, map[string]any{"op": "replace", "path": "/spec/template", "value": op.Template})
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}
	options := metav1.PatchOptions{FieldManager: "kubevista-durable"}
	if dry {
		options.DryRun = []string{metav1.DryRunAll}
	}
	return c.client.AppsV1().Deployments(op.Request.Namespace).Patch(ctx, op.Request.Name, types.JSONPatchType, payload, options)
}
