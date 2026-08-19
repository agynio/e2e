//go:build e2e && (svc_agents_orchestrator || svc_runners || svc_metering || svc_k8s_runner || svc_organizations || svc_llm || svc_llm_proxy || svc_images || smoke)

package tests

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func logTracingDiagnostics(t *testing.T, threadID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	clientset, ok := diagnosticsClientset(t)
	if !ok {
		return
	}
	namespace := workloadNamespace(t)
	selector := fmt.Sprintf("%s=%s,%s=%s", labelManagedBy, managedByValue, labelThreadID, threadID)
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		t.Logf("diagnostics: list workload pods: %v", err)
		return
	}
	if len(pods.Items) == 0 {
		t.Logf("diagnostics: no workload pods found for thread %s", threadID)
		return
	}
	for _, pod := range pods.Items {
		logWorkloadPodDiagnosticsFromPod(t, ctx, pod)
	}
}

type logReadOptions struct {
	TailLines int64
	MaxLines  int
	Previous  bool
}

func readWorkloadLogs(t *testing.T, ctx context.Context, namespace, podName, containerName string) {
	t.Helper()
	// The agent container is the one that explains a failed workload, so give
	// it room; five lines is not enough to show a startup sequence and its
	// error.
	options := logReadOptions{TailLines: 200, MaxLines: 25}
	if strings.HasPrefix(containerName, "mcp-") {
		options.MaxLines = 20
	}
	readWorkloadLogsWithOptions(t, ctx, namespace, podName, containerName, options)
}

func readWorkloadLogsWithOptions(t *testing.T, ctx context.Context, namespace, podName, containerName string, options logReadOptions) {
	t.Helper()
	clientset, ok := diagnosticsClientset(t)
	if !ok {
		return
	}
	request := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
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

	// Keep the LAST MaxLines of the window, not the first. A container that
	// died explains itself on its final line, and printing the head of the
	// tail window hides exactly that — a crash reads as a silent exit.
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	recent := make([]string, 0, options.MaxLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if len(recent) == options.MaxLines {
			recent = recent[1:]
		}
		recent = append(recent, line)
	}
	if err := scanner.Err(); err != nil {
		t.Logf("diagnostics: pod=%s container=%s log scan error: %v", podName, containerName, err)
	}
	for _, line := range recent {
		t.Logf("diagnostics: pod=%s container=%s log=%s", podName, containerName, truncateLogLine(line))
	}
}

func logTracingStackDiagnostics(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	namespace := currentNamespace(t)
	clientset, ok := diagnosticsClientset(t)
	if !ok {
		return
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Logf("diagnostics: list tracing pods: %v", err)
		return
	}
	found := 0
	for _, pod := range pods.Items {
		if !isTracingPod(pod) {
			continue
		}
		t.Logf("diagnostics: tracing pod=%s", pod.Name)
		for _, container := range pod.Spec.Containers {
			t.Logf("diagnostics: tracing pod=%s container=%s", pod.Name, container.Name)
			readWorkloadLogs(t, ctx, namespace, pod.Name, container.Name)
		}
		found++
		if found >= 2 {
			break
		}
	}
	if found == 0 {
		t.Log("diagnostics: no tracing pods found")
	}
}

func isTracingPod(pod corev1.Pod) bool {
	name := strings.ToLower(pod.Name)
	if strings.Contains(name, "tracing") || strings.Contains(name, "tempo") || strings.Contains(name, "otel") || strings.Contains(name, "collector") {
		return true
	}
	for key, value := range pod.Labels {
		labelKey := strings.ToLower(key)
		labelValue := strings.ToLower(value)
		if strings.Contains(labelKey, "tracing") || strings.Contains(labelValue, "tracing") {
			return true
		}
		if strings.Contains(labelKey, "otel") || strings.Contains(labelValue, "otel") {
			return true
		}
		if strings.Contains(labelKey, "tempo") || strings.Contains(labelValue, "tempo") {
			return true
		}
		if strings.Contains(labelKey, "collector") || strings.Contains(labelValue, "collector") {
			return true
		}
	}
	return false
}

func truncateLogLine(line string) string {
	if line == "" {
		return line
	}
	lineRunes := []rune(line)
	if len(lineRunes) <= 1000 {
		return line
	}
	return string(lineRunes[:1000])
}

func logWorkloadPodDiagnostics(t *testing.T, ctx context.Context, workloadID string) {
	t.Helper()
	namespace := workloadNamespace(t)
	podName := fmt.Sprintf("workload-%s", workloadID)
	clientset, ok := diagnosticsClientset(t)
	if !ok {
		return
	}
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		t.Logf("diagnostics: pod=%s get error: %v", podName, err)
		return
	}
	logWorkloadPodDiagnosticsFromPod(t, ctx, *pod)
}

// logLLMProxyDiagnostics dumps the proxy's own account of a turn. An agent that
// reports "stream disconnected before completion" only sees its end of the
// connection; the proxy logs the upstream status and any stream failure, and
// without it a transport fault is indistinguishable from the agent misbehaving.
func logLLMProxyDiagnostics(t *testing.T, ctx context.Context) {
	t.Helper()
	namespace := currentNamespace(t)
	clientset, ok := diagnosticsClientset(t)
	if !ok {
		return
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Logf("diagnostics: list llm-proxy pods: %v", err)
		return
	}
	found := 0
	for _, pod := range pods.Items {
		if !strings.Contains(strings.ToLower(pod.Name), "llm-proxy") {
			continue
		}
		t.Logf("diagnostics: llm-proxy pod=%s", pod.Name)
		for _, container := range pod.Spec.Containers {
			readWorkloadLogsWithOptions(t, ctx, namespace, pod.Name, container.Name,
				logReadOptions{TailLines: 200, MaxLines: 25})
		}
		found++
		if found >= 2 {
			break
		}
	}
	if found == 0 {
		t.Log("diagnostics: no llm-proxy pods found")
	}
}

func logWorkloadPodDiagnosticsFromPod(t *testing.T, ctx context.Context, pod corev1.Pod) {
	t.Helper()
	logWorkloadPodStatus(t, pod)
	logLLMProxyDiagnostics(t, ctx)
	namespace := pod.Namespace
	if namespace == "" {
		namespace = workloadNamespace(t)
	}
	logWorkloadPodEvents(t, ctx, namespace, pod.Name)
	logWorkloadInitContainerLogs(t, ctx, namespace, pod)
	logWorkloadContainerLogs(t, ctx, namespace, pod)
}

func logWorkloadPodStatus(t *testing.T, pod corev1.Pod) {
	t.Helper()
	message := truncateLogLine(pod.Status.Message)
	if message == "" {
		message = "-"
	}
	t.Logf(
		"diagnostics: pod=%s phase=%s reason=%s message=%s",
		pod.Name,
		pod.Status.Phase,
		pod.Status.Reason,
		message,
	)
	for _, condition := range pod.Status.Conditions {
		conditionMessage := truncateLogLine(condition.Message)
		if conditionMessage == "" {
			conditionMessage = "-"
		}
		t.Logf(
			"diagnostics: pod=%s condition=%s status=%s reason=%s message=%s",
			pod.Name,
			condition.Type,
			condition.Status,
			condition.Reason,
			conditionMessage,
		)
	}
	for _, status := range pod.Status.InitContainerStatuses {
		logContainerStatus(t, pod.Name, "init", status)
	}
	for _, status := range pod.Status.ContainerStatuses {
		logContainerStatus(t, pod.Name, "container", status)
	}
}

func logContainerStatus(t *testing.T, podName, kind string, status corev1.ContainerStatus) {
	t.Helper()
	state, reason, message, exitCode := summarizeContainerState(status.State)
	if message == "" {
		message = "-"
	}
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
		truncateLogLine(message),
	)
	if status.LastTerminationState.Terminated != nil {
		last := status.LastTerminationState.Terminated
		lastMessage := truncateLogLine(last.Message)
		if lastMessage == "" {
			lastMessage = "-"
		}
		t.Logf(
			"diagnostics: pod=%s %s=%s last_exit=%d last_reason=%s last_message=%s",
			podName,
			kind,
			status.Name,
			last.ExitCode,
			last.Reason,
			lastMessage,
		)
	}
}

func summarizeContainerState(state corev1.ContainerState) (string, string, string, int32) {
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

func logWorkloadPodEvents(t *testing.T, ctx context.Context, namespace, podName string) {
	t.Helper()
	clientset, ok := diagnosticsClientset(t)
	if !ok {
		return
	}
	fieldSelector := fmt.Sprintf("involvedObject.name=%s", podName)
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		t.Logf("diagnostics: pod=%s events error: %v", podName, err)
		return
	}
	if len(events.Items) == 0 {
		t.Logf("diagnostics: pod=%s no events", podName)
		return
	}
	sort.Slice(events.Items, func(i, j int) bool {
		return eventTimestamp(events.Items[i]).Before(eventTimestamp(events.Items[j]))
	})
	start := len(events.Items) - 5
	if start < 0 {
		start = 0
	}
	for _, event := range events.Items[start:] {
		eventMessage := truncateLogLine(event.Message)
		if eventMessage == "" {
			eventMessage = "-"
		}
		t.Logf(
			"diagnostics: pod=%s event=%s reason=%s type=%s message=%s",
			podName,
			event.Name,
			event.Reason,
			event.Type,
			eventMessage,
		)
	}
}

func eventTimestamp(event corev1.Event) time.Time {
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	return time.Time{}
}

func logWorkloadContainerLogs(t *testing.T, ctx context.Context, namespace string, pod corev1.Pod) {
	t.Helper()
	containers := workloadContainerNames(pod)
	if len(containers) == 0 {
		t.Logf("diagnostics: pod=%s no agent/mcp containers found", pod.Name)
		return
	}
	for _, container := range containers {
		t.Logf("diagnostics: workload pod=%s container=%s", pod.Name, container)
		readWorkloadLogs(t, ctx, namespace, pod.Name, container)
	}
}

func logWorkloadInitContainerLogs(t *testing.T, ctx context.Context, namespace string, pod corev1.Pod) {
	t.Helper()
	for _, container := range pod.Spec.InitContainers {
		if container.Name != "ziti-sidecar" {
			continue
		}
		t.Logf("diagnostics: workload pod=%s init-container=%s", pod.Name, container.Name)
		readWorkloadLogsWithOptions(t, ctx, namespace, pod.Name, container.Name, logReadOptions{
			TailLines: 200,
			MaxLines:  20,
		})
		t.Logf("diagnostics: workload pod=%s init-container=%s (previous)", pod.Name, container.Name)
		readWorkloadLogsWithOptions(t, ctx, namespace, pod.Name, container.Name, logReadOptions{
			TailLines: 200,
			MaxLines:  20,
			Previous:  true,
		})
	}
}

func workloadContainerNames(pod corev1.Pod) []string {
	containers := make([]string, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		name := container.Name
		if strings.HasPrefix(name, "agent-") || strings.HasPrefix(name, "mcp-") {
			containers = append(containers, name)
		}
	}
	return containers
}
