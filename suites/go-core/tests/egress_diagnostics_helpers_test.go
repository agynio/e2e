//go:build e2e && svc_egress_gateway && !(svc_k8s_runner || smoke)

package tests

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type egressLogReadOptions struct {
	TailLines int64
	MaxLines  int
	Previous  bool
}

func logEgressWorkloadPodDiagnostics(t *testing.T, ctx context.Context, workloadID string) {
	t.Helper()
	namespace := egressWorkloadNamespace(t)
	podName := fmt.Sprintf("workload-%s", workloadID)
	clientset := egressKubeClientset(t)
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Logf("diagnostics: pod=%s get error: %v", podName, err)
		return
	}
	logEgressWorkloadPodStatus(t, *pod)
	for _, container := range pod.Spec.InitContainers {
		t.Logf("diagnostics: workload pod=%s init-container=%s", pod.Name, container.Name)
		readEgressWorkloadLogs(t, ctx, namespace, pod.Name, container.Name, egressLogReadOptions{TailLines: 200, MaxLines: 20})
		t.Logf("diagnostics: workload pod=%s init-container=%s (previous)", pod.Name, container.Name)
		readEgressWorkloadLogs(t, ctx, namespace, pod.Name, container.Name, egressLogReadOptions{TailLines: 200, MaxLines: 20, Previous: true})
	}
	for _, container := range pod.Spec.Containers {
		t.Logf("diagnostics: workload pod=%s container=%s", pod.Name, container.Name)
		readEgressWorkloadLogs(t, ctx, namespace, pod.Name, container.Name, egressLogReadOptions{TailLines: 100, MaxLines: 10})
	}
}

func logEgressWorkloadPodStatus(t *testing.T, pod corev1.Pod) {
	t.Helper()
	t.Logf("diagnostics: pod=%s phase=%s reason=%s message=%s", pod.Name, pod.Status.Phase, pod.Status.Reason, truncateEgressLogLine(pod.Status.Message))
	for _, status := range pod.Status.InitContainerStatuses {
		logEgressContainerStatus(t, pod.Name, "init", status)
	}
	for _, status := range pod.Status.ContainerStatuses {
		logEgressContainerStatus(t, pod.Name, "container", status)
	}
}

func logEgressContainerStatus(t *testing.T, podName, kind string, status corev1.ContainerStatus) {
	t.Helper()
	state, reason, message, exitCode := egressContainerState(status.State)
	t.Logf(
		"diagnostics: pod=%s %s=%s ready=%t restarts=%d state=%s reason=%s exit=%d message=%s",
		podName,
		kind,
		status.Name,
		status.Ready,
		status.RestartCount,
		state,
		reason,
		exitCode,
		truncateEgressLogLine(message),
	)
}

func egressContainerState(state corev1.ContainerState) (string, string, string, int32) {
	switch {
	case state.Running != nil:
		return "running", "", "", 0
	case state.Waiting != nil:
		return "waiting", state.Waiting.Reason, state.Waiting.Message, 0
	case state.Terminated != nil:
		return "terminated", state.Terminated.Reason, state.Terminated.Message, state.Terminated.ExitCode
	default:
		return "unknown", "", "", 0
	}
}

func readEgressWorkloadLogs(t *testing.T, ctx context.Context, namespace, podName, containerName string, options egressLogReadOptions) {
	t.Helper()
	request := egressKubeClientset(t).CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:  containerName,
		TailLines:  &options.TailLines,
		Timestamps: true,
		Previous:   options.Previous,
	})
	stream, err := request.Stream(ctx)
	if err != nil {
		t.Logf("diagnostics: pod=%s container=%s log error: %v", podName, containerName, err)
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		t.Logf("diagnostics: pod=%s container=%s log=%s", podName, containerName, truncateEgressLogLine(line))
		lines++
		if lines >= options.MaxLines {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		t.Logf("diagnostics: pod=%s container=%s log scan error: %v", podName, containerName, err)
	}
}

func truncateEgressLogLine(line string) string {
	if line == "" {
		return "-"
	}
	lineRunes := []rune(line)
	if len(lineRunes) <= 1000 {
		return line
	}
	return string(lineRunes[:1000])
}
