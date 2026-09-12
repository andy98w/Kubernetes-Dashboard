package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestDiagnosisContainerFailures(t *testing.T) {
	for _, reason := range []string{"CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "CreateContainerConfigError"} {
		t.Run(reason, func(t *testing.T) {
			pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "broken"}, Status: corev1.PodStatus{InitContainerStatuses: []corev1.ContainerStatus{{Name: "init", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: "actual evidence"}}}}}}
			got := diagnosePods([]corev1.Pod{pod})
			if len(got) != 1 || got[0].Code != reason || got[0].Evidence != "actual evidence" || got[0].NextStep == "" {
				t.Fatalf("unexpected diagnosis: %+v", got)
			}
			now := metav1.Now()
			pod.DeletionTimestamp = &now
			if len(diagnosePods([]corev1.Pod{pod})) != 0 {
				t.Fatal("terminating pod should not trigger diagnosis")
			}
		})
	}
}

func TestDiagnosisHistoricalOOMAndScheduling(t *testing.T) {
	pod := corev1.Pod{Status: corev1.PodStatus{
		Conditions:        []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Message: "Insufficient memory"}},
		ContainerStatuses: []corev1.ContainerStatus{{Name: "app", Ready: true, LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137}}}},
	}}
	got := diagnosePods([]corev1.Pod{pod})
	if len(got) != 2 || got[0].Code != "Unschedulable" || got[1].Code != "OOMKilled" {
		t.Fatalf("unexpected: %+v", got)
	}
	if len(diagnosePods([]corev1.Pod{{}})) != 0 {
		t.Fatal("empty status must not invent a diagnosis")
	}
}

func TestDiagnosisProbeEvidence(t *testing.T) {
	got := diagnoseEvents([]Event{{Type: "Warning", Reason: "Unhealthy", Message: "Readiness probe failed: HTTP 404"}, {Type: "Normal", Reason: "Unhealthy", Message: "probe"}, {Type: "Warning", Reason: "BackOff", Message: "retry"}})
	if len(got) != 1 || got[0].Code != "ProbeFailed" {
		t.Fatalf("unexpected: %+v", got)
	}
}
