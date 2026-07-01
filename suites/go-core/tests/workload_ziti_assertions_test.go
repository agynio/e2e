//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"fmt"
	"testing"

	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	zitiEnrollContainerName      = "ziti-enroll"
	zitiGatewayWaitContainerName = "ziti-gateway-wait"
	zitiSidecarContainerName     = "ziti-sidecar"
)

func assertWorkloadZitiSidecarReady(
	t *testing.T,
	ctx context.Context,
	runnerClient runnerv1.RunnerServiceClient,
	labels map[string]string,
) {
	t.Helper()
	workloadID, pod, err := waitForWorkloadZitiSidecarReady(t, ctx, runnerClient, labels)
	if err == nil {
		t.Logf("validated Ziti sidecar for workload=%s pod=%s", workloadID, pod.Name)
		return
	}
	logWorkloadsForLabelsDiagnostics(t, ctx, runnerClient, labels)
	t.Fatalf("validate Ziti sidecar for real-agent workload: %v", err)
}

func waitForWorkloadZitiSidecarReady(
	t *testing.T,
	ctx context.Context,
	runnerClient runnerv1.RunnerServiceClient,
	labels map[string]string,
) (string, *corev1.Pod, error) {
	t.Helper()
	var readyWorkloadID string
	var readyPod *corev1.Pod
	err := pollUntil(ctx, pollInterval, func(ctx context.Context) error {
		workloadIDs, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			return err
		}
		if len(workloadIDs) == 0 {
			return fmt.Errorf("no real-agent workload pods found for labels %v", labels)
		}
		for _, workloadID := range workloadIDs {
			pod, err := getWorkloadPod(t, ctx, workloadID)
			if err != nil {
				return err
			}
			if err := validateWorkloadZitiSidecarPod(pod); err != nil {
				return err
			}
			readyWorkloadID = workloadID
			readyPod = pod
			return nil
		}
		return fmt.Errorf("no workload IDs returned for labels %v", labels)
	})
	if err != nil {
		return "", nil, err
	}
	return readyWorkloadID, readyPod, nil
}

func getWorkloadPod(t *testing.T, ctx context.Context, workloadID string) (*corev1.Pod, error) {
	t.Helper()
	podName := fmt.Sprintf("workload-%s", workloadID)
	pod, err := kubeClientset(t).CoreV1().Pods(workloadNamespace(t)).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get workload pod %s: %w", podName, err)
	}
	return pod, nil
}

func validateWorkloadZitiSidecarPod(pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("workload pod is nil")
	}
	if pod.Name == "" {
		return fmt.Errorf("workload pod name is empty")
	}
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("workload pod %s phase=%s", pod.Name, pod.Status.Phase)
	}
	if err := requireInitContainerSucceeded(pod, zitiEnrollContainerName); err != nil {
		return err
	}
	if err := requireInitContainerSucceeded(pod, zitiGatewayWaitContainerName); err != nil {
		return err
	}
	if err := requireRestartableInitContainerRunning(pod, zitiSidecarContainerName); err != nil {
		return err
	}
	return nil
}

func requireInitContainerSucceeded(pod *corev1.Pod, name string) error {
	if !hasInitContainerSpec(pod, name) {
		return fmt.Errorf("workload pod %s missing init container %s", pod.Name, name)
	}
	status, ok := initContainerStatusByName(pod, name)
	if !ok {
		return fmt.Errorf("workload pod %s missing status for init container %s", pod.Name, name)
	}
	terminated := status.State.Terminated
	if terminated == nil || terminated.ExitCode != 0 {
		state, reason, message, exitCode := summarizeContainerState(status.State)
		return fmt.Errorf(
			"workload pod %s init container %s not completed successfully: state=%s reason=%s exit=%d message=%s",
			pod.Name,
			name,
			state,
			reason,
			exitCode,
			truncateLogLine(message),
		)
	}
	return nil
}

func requireRestartableInitContainerRunning(pod *corev1.Pod, name string) error {
	if !hasInitContainerSpec(pod, name) && !hasAppContainerSpec(pod, name) {
		return fmt.Errorf("workload pod %s missing Ziti sidecar container %s", pod.Name, name)
	}
	if status, ok := initContainerStatusByName(pod, name); ok {
		if status.State.Running != nil && status.Ready {
			return nil
		}
		state, reason, message, exitCode := summarizeContainerState(status.State)
		return fmt.Errorf(
			"workload pod %s restartable init container %s not running/ready: state=%s reason=%s exit=%d message=%s",
			pod.Name,
			name,
			state,
			reason,
			exitCode,
			truncateLogLine(message),
		)
	}
	if status, ok := appContainerStatusByName(pod, name); ok {
		if status.State.Running != nil && status.Ready {
			return nil
		}
		state, reason, message, exitCode := summarizeContainerState(status.State)
		return fmt.Errorf(
			"workload pod %s sidecar container %s not running/ready: state=%s reason=%s exit=%d message=%s",
			pod.Name,
			name,
			state,
			reason,
			exitCode,
			truncateLogLine(message),
		)
	}
	return fmt.Errorf("workload pod %s missing status for Ziti sidecar container %s", pod.Name, name)
}

func hasInitContainerSpec(pod *corev1.Pod, name string) bool {
	for _, container := range pod.Spec.InitContainers {
		if container.Name == name {
			return true
		}
	}
	return false
}

func hasAppContainerSpec(pod *corev1.Pod, name string) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == name {
			return true
		}
	}
	return false
}

func initContainerStatusByName(pod *corev1.Pod, name string) (corev1.ContainerStatus, bool) {
	for _, status := range pod.Status.InitContainerStatuses {
		if status.Name == name {
			return status, true
		}
	}
	return corev1.ContainerStatus{}, false
}

func appContainerStatusByName(pod *corev1.Pod, name string) (corev1.ContainerStatus, bool) {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == name {
			return status, true
		}
	}
	return corev1.ContainerStatus{}, false
}

func logWorkloadsForLabelsDiagnostics(
	t *testing.T,
	ctx context.Context,
	runnerClient runnerv1.RunnerServiceClient,
	labels map[string]string,
) {
	t.Helper()
	workloadIDs, err := findWorkloadsByLabels(ctx, runnerClient, labels)
	if err != nil {
		t.Logf("diagnostics: find workloads for Ziti sidecar assertion: %v", err)
		return
	}
	if len(workloadIDs) == 0 {
		t.Logf("diagnostics: no real-agent workloads found for labels %v", labels)
		return
	}
	for _, workloadID := range workloadIDs {
		logWorkloadPodDiagnostics(t, ctx, workloadID)
	}
}
