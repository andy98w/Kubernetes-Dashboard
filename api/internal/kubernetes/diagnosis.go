package kubernetes

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Diagnosis reports observed symptoms, not an inferred root cause.
type Diagnosis struct {
	Code     string `json:"code"`
	Resource string `json:"resource"`
	Evidence string `json:"evidence"`
	NextStep string `json:"nextStep"`
}

func diagnosePods(pods []corev1.Pod) []Diagnosis {
	findings := []Diagnosis{}
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
				findings = append(findings, Diagnosis{"Unschedulable", "Pod/" + pod.Name, condition.Message, "Check the scheduler message for insufficient capacity, taints, affinity, or an unbound volume. Scaling replicas will not add node capacity."})
			}
		}
		statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for _, status := range statuses {
			resource := "Pod/" + pod.Name + " · " + status.Name
			if waiting := status.State.Waiting; waiting != nil {
				next := ""
				switch waiting.Reason {
				case "CrashLoopBackOff":
					next = "Inspect the previous container logs and exit code. Correct the process or configuration before restarting; a restart alone may repeat the failure."
				case "ImagePullBackOff", "ErrImagePull":
					next = "Verify the image reference, registry access, and image pull credentials. Restarting does not correct a missing image."
				case "CreateContainerConfigError":
					next = "Check referenced ConfigMaps and Secrets exist in this namespace. Do not include secret values in incident reports."
				}
				if next != "" {
					findings = append(findings, Diagnosis{waiting.Reason, resource, waiting.Message, next})
				}
			}
			terminated := status.State.Terminated
			if terminated == nil {
				terminated = status.LastTerminationState.Terminated
			}
			if terminated != nil && terminated.Reason == "OOMKilled" {
				findings = append(findings, Diagnosis{"OOMKilled", resource, fmt.Sprintf("Container was OOMKilled (exit %d, restart count %d). This may be a previous termination; check current readiness.", terminated.ExitCode, status.RestartCount), "Compare memory usage with the container limit and investigate growth or leaks before increasing the limit."})
			}
		}
	}
	return findings
}

func diagnoseEvents(events []Event) []Diagnosis {
	findings := []Diagnosis{}
	for _, event := range events {
		if event.Type == "Warning" && event.Reason == "Unhealthy" && strings.Contains(strings.ToLower(event.Message), "probe") {
			findings = append(findings, Diagnosis{"ProbeFailed", event.Object, event.Message, "Check the probe path, port, timeout, and application startup. This retained event may predate recovery; compare it with current pod readiness."})
		}
	}
	return findings
}
