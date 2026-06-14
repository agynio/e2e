//go:build e2e && svc_egress_gateway && !(svc_k8s_runner || smoke)

package tests

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type egressLogReadOptions struct {
	TailLines int64
	MaxLines  int
	Previous  bool
}

type egressEnrollmentClaims struct {
	Issuer             string   `json:"iss"`
	Subject            string   `json:"sub"`
	Audience           []string `json:"aud"`
	ExpiresAt          int64    `json:"exp"`
	JWTID              string   `json:"jti"`
	EnrollmentMethod   string   `json:"em"`
	ControllerEndpoint []string `json:"ctrls"`
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

func logEgressEnrollmentDiagnostics(t *testing.T, ctx context.Context, zitiIdentityID, enrollmentJWT string) {
	t.Helper()
	claims, err := parseEgressEnrollmentClaims(enrollmentJWT)
	if err != nil {
		t.Logf("diagnostics: ziti enrollment jwt parse error: %v", err)
	} else {
		t.Logf(
			"diagnostics: ziti enrollment jwt identity_id=%s iss=%s sub=%s aud=%s exp=%s jti=%s em=%s ctrls=%s",
			zitiIdentityID,
			egressDiagnosticValue(claims.Issuer),
			egressDiagnosticValue(claims.Subject),
			egressDiagnosticValue(strings.Join(claims.Audience, ",")),
			formatEgressUnixTime(claims.ExpiresAt),
			egressDiagnosticValue(claims.JWTID),
			egressDiagnosticValue(claims.EnrollmentMethod),
			egressDiagnosticValue(strings.Join(claims.ControllerEndpoint, ",")),
		)
	}

	session, err := createEgressZitiManagementSession(t, ctx)
	if err != nil {
		t.Logf("diagnostics: ziti management unavailable: %v", err)
		return
	}
	logEgressZitiResource(t, ctx, session, "identity-by-id", "/identities/"+url.PathEscape(zitiIdentityID))
	logEgressZitiResource(t, ctx, session, "identity-enrollments", "/identities/"+url.PathEscape(zitiIdentityID)+"/enrollments")
	if claims != nil && claims.JWTID != "" {
		logEgressZitiResource(t, ctx, session, "enrollment-by-token", egressZitiFilterPath("/enrollments", "token", claims.JWTID))
	}
}

func parseEgressEnrollmentClaims(enrollmentJWT string) (*egressEnrollmentClaims, error) {
	parts := strings.Split(strings.TrimSpace(enrollmentJWT), ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("expected 3 jwt segments, got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var claims egressEnrollmentClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	return &claims, nil
}

func formatEgressUnixTime(unixSeconds int64) string {
	if unixSeconds == 0 {
		return "-"
	}
	return time.Unix(unixSeconds, 0).UTC().Format(time.RFC3339)
}

func egressDiagnosticValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

type egressZitiManagementSession struct {
	endpoint string
	token    string
	client   *http.Client
}

type egressZitiAuthenticationResponse struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

func createEgressZitiManagementSession(t *testing.T, ctx context.Context) (egressZitiManagementSession, error) {
	t.Helper()
	username, password, err := egressZitiDiagnosticsCredentials(t, ctx)
	if err != nil {
		return egressZitiManagementSession{}, err
	}

	endpoint := egressZitiManagementEndpoint()
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	payload := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/authenticate?method=password", strings.NewReader(payload))
	if err != nil {
		return egressZitiManagementSession{}, fmt.Errorf("build authenticate request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return egressZitiManagementSession{}, fmt.Errorf("authenticate: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return egressZitiManagementSession{}, fmt.Errorf("read authenticate response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return egressZitiManagementSession{}, fmt.Errorf("authenticate status %d body=%s", response.StatusCode, truncateEgressLogLine(strings.TrimSpace(string(body))))
	}

	var auth egressZitiAuthenticationResponse
	if err := json.Unmarshal(body, &auth); err != nil {
		return egressZitiManagementSession{}, fmt.Errorf("parse authenticate response: %w", err)
	}
	token := strings.TrimSpace(auth.Data.Token)
	if token == "" {
		return egressZitiManagementSession{}, fmt.Errorf("authenticate response missing token")
	}
	return egressZitiManagementSession{endpoint: endpoint, token: token, client: client}, nil
}

func egressZitiDiagnosticsCredentials(t *testing.T, ctx context.Context) (string, string, error) {
	t.Helper()
	secret, err := egressKubeClientset(t).CoreV1().Secrets(platformNamespace()).Get(ctx, "ziti-management-diagnostics", metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("get %s/%s: %w", platformNamespace(), "ziti-management-diagnostics", err)
	}
	username := strings.TrimSpace(string(secret.Data["username"]))
	password := strings.TrimSpace(string(secret.Data["password"]))
	if username == "" || password == "" {
		return "", "", fmt.Errorf("%s/%s missing diagnostics credentials", platformNamespace(), "ziti-management-diagnostics")
	}
	return username, password, nil
}

func egressZitiManagementEndpoint() string {
	if explicitEndpoint := strings.TrimSpace(envOrDefault("ZITI_MGMT_ENDPOINT", "")); explicitEndpoint != "" {
		return strings.TrimRight(explicitEndpoint, "/")
	}
	domain := envOrDefault("E2E_DOMAIN", envOrDefault("DOMAIN", "agyn.dev"))
	port := envOrDefault("E2E_INGRESS_PORT", envOrDefault("INGRESS_PORT", envOrDefault("PORT", "2496")))
	return strings.TrimRight(fmt.Sprintf("https://ziti-mgmt.%s:%s/edge/management/v1", domain, port), "/")
}

func egressZitiFilterPath(resourcePath, field, value string) string {
	return fmt.Sprintf("%s?filter=%s%%3D%%22%s%%22", resourcePath, url.QueryEscape(field), url.QueryEscape(value))
}

func logEgressZitiResource(t *testing.T, ctx context.Context, session egressZitiManagementSession, label, path string) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, session.endpoint+path, nil)
	if err != nil {
		t.Logf("diagnostics: ziti %s request error: %v", label, err)
		return
	}
	request.Header.Set("zt-session", session.token)

	response, err := session.client.Do(request)
	if err != nil {
		t.Logf("diagnostics: ziti %s query error: %v", label, err)
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Logf("diagnostics: ziti %s read error: %v", label, err)
		return
	}
	trimmedBody := strings.TrimSpace(string(body))
	if trimmedBody == "" {
		trimmedBody = "{}"
	}
	t.Logf("diagnostics: ziti %s status=%d body=%s", label, response.StatusCode, truncateEgressLogLine(trimmedBody))
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
