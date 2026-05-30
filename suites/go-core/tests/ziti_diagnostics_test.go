//go:build e2e && (svc_agents_orchestrator || svc_runners || svc_metering || svc_k8s_runner || svc_organizations || svc_files || svc_gateway || svc_llm || svc_llm_proxy || smoke)

package tests

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	zitiDiagnosticsTimeout        = 20 * time.Second
	zitiDiagnosticsRequestTimeout = 15 * time.Second
	zitiMgmtEndpointEnvKey        = "ZITI_MGMT_ENDPOINT"
	zitiMgmtServiceName           = "ziti-mgmt"
	zitiMgmtPath                  = "/edge/management/v1"
	// DEV/E2E ONLY: ziti-diagnostics must only exist in dev/E2E bootstrap
	// deployments and must never be enabled in production.
	zitiDiagnosticsSecretName  = "ziti-diagnostics"
	zitiDiagnosticsUserKey     = "username"
	zitiDiagnosticsPasswordKey = "password"
)

type zitiDiagnosticsQuery struct {
	Label string
	Path  string
}

type zitiManagementSession struct {
	endpoint string
	token    string
	client   *http.Client
}

type zitiAuthenticationResponse struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

func TestZitiManagementEndpointDefaultUsesIngress(t *testing.T) {
	t.Setenv(zitiMgmtEndpointEnvKey, "")
	t.Setenv("E2E_DOMAIN", "e2e.agyn.dev")
	t.Setenv("E2E_INGRESS_PORT", "30443")

	got := zitiManagementEndpoint()
	want := "https://ziti-mgmt.e2e.agyn.dev:30443/edge/management/v1"
	if got != want {
		t.Fatalf("ziti management endpoint mismatch: got %q want %q", got, want)
	}
}

func TestZitiManagementEndpointExplicitOverride(t *testing.T) {
	t.Setenv(zitiMgmtEndpointEnvKey, "https://custom.example.test:9443/edge/management/v1/")

	got := zitiManagementEndpoint()
	want := "https://custom.example.test:9443/edge/management/v1"
	if got != want {
		t.Fatalf("ziti management endpoint mismatch: got %q want %q", got, want)
	}
}

func TestZitiDiagnosticsSecretUsesPlatformNamespace(t *testing.T) {
	secretNamespace, secretName := zitiDiagnosticsSecretRef()
	if secretNamespace != "platform" {
		t.Fatalf("ziti diagnostics secret namespace mismatch: got %q want %q", secretNamespace, "platform")
	}
	if secretName != zitiDiagnosticsSecretName {
		t.Fatalf("ziti diagnostics secret name mismatch: got %q want %q", secretName, zitiDiagnosticsSecretName)
	}
}

func TestZitiDiagnosticsSecretUsesDevspaceNamespace(t *testing.T) {
	t.Setenv("DEVSPACE_NAMESPACE", "custom-platform")
	secretNamespace, secretName := zitiDiagnosticsSecretRef()
	if secretNamespace != "custom-platform" {
		t.Fatalf("ziti diagnostics secret namespace mismatch: got %q want %q", secretNamespace, "custom-platform")
	}
	if secretName != zitiDiagnosticsSecretName {
		t.Fatalf("ziti diagnostics secret name mismatch: got %q want %q", secretName, zitiDiagnosticsSecretName)
	}
}

func TestZitiDiagnosticsDump(t *testing.T) {
	if strings.TrimSpace(envOrDefault("E2E_ZITI_DIAGNOSTICS_DUMP", "")) != "1" {
		t.Skip("set E2E_ZITI_DIAGNOSTICS_DUMP=1 to dump shared Ziti diagnostics")
	}
	ctx, cancel := context.WithTimeout(context.Background(), zitiDiagnosticsTimeout)
	defer cancel()
	logZitiDiagnostics(t, ctx, nil)
}

func registerZitiFailureDiagnostics(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), zitiDiagnosticsTimeout)
		defer cancel()
		logZitiDiagnostics(t, ctx, nil)
	})
}

func logZitiDiagnostics(t *testing.T, ctx context.Context, queries []zitiDiagnosticsQuery) {
	t.Helper()
	session, err := createZitiManagementSession(t, ctx)
	if err != nil {
		t.Logf("diagnostics: ziti management unavailable: %v", err)
		return
	}
	if len(queries) == 0 {
		queries = defaultZitiDiagnosticsQueries()
	}
	for _, query := range queries {
		logZitiResource(t, ctx, session, query.Label, query.Path)
	}
}

func defaultZitiDiagnosticsQueries() []zitiDiagnosticsQuery {
	return []zitiDiagnosticsQuery{
		{Label: "services", Path: "/services?limit=100"},
		{Label: "terminators", Path: "/terminators?limit=100"},
		{Label: "service-policies", Path: "/service-policies?limit=100"},
		{Label: "identities", Path: "/identities?limit=100"},
	}
}

func zitiFilterPath(resourcePath, field, value string) string {
	return fmt.Sprintf("%s?filter=%s%%3D%%22%s%%22", resourcePath, url.QueryEscape(field), url.QueryEscape(value))
}

func createZitiManagementSession(t *testing.T, ctx context.Context) (zitiManagementSession, error) {
	t.Helper()
	username, password, err := zitiDiagnosticsCredentials(t, ctx)
	if err != nil {
		return zitiManagementSession{}, err
	}

	endpoint := zitiManagementEndpoint()
	client := &http.Client{
		Timeout:   zitiDiagnosticsRequestTimeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	payload := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/authenticate?method=password", strings.NewReader(payload))
	if err != nil {
		return zitiManagementSession{}, fmt.Errorf("build authenticate request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return zitiManagementSession{}, fmt.Errorf("authenticate: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return zitiManagementSession{}, fmt.Errorf("read authenticate response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return zitiManagementSession{}, fmt.Errorf("authenticate status %d body=%s", response.StatusCode, truncateLogLine(strings.TrimSpace(string(body))))
	}

	var auth zitiAuthenticationResponse
	if err := json.Unmarshal(body, &auth); err != nil {
		return zitiManagementSession{}, fmt.Errorf("parse authenticate response: %w", err)
	}
	token := strings.TrimSpace(auth.Data.Token)
	if token == "" {
		return zitiManagementSession{}, fmt.Errorf("authenticate response missing token")
	}
	return zitiManagementSession{endpoint: endpoint, token: token, client: client}, nil
}

func zitiDiagnosticsCredentials(t *testing.T, ctx context.Context) (string, string, error) {
	t.Helper()
	secretNamespace, secretName := zitiDiagnosticsSecretRef()
	secret, err := kubeClientset(t).CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("get %s/%s: %w", secretNamespace, secretName, err)
	}
	username := strings.TrimSpace(string(secret.Data[zitiDiagnosticsUserKey]))
	password := strings.TrimSpace(string(secret.Data[zitiDiagnosticsPasswordKey]))
	if username == "" || password == "" {
		return "", "", fmt.Errorf("%s/%s missing diagnostics credentials", secretNamespace, secretName)
	}
	return username, password, nil
}

func zitiDiagnosticsSecretRef() (string, string) {
	return envOrDefault("E2E_NAMESPACE", envOrDefault("DEVSPACE_NAMESPACE", "platform")), zitiDiagnosticsSecretName
}

func zitiManagementEndpoint() string {
	if explicitEndpoint := strings.TrimSpace(envOrDefault(zitiMgmtEndpointEnvKey, "")); explicitEndpoint != "" {
		return strings.TrimRight(explicitEndpoint, "/")
	}
	domain := envOrDefault("E2E_DOMAIN", envOrDefault("DOMAIN", "agyn.dev"))
	port := envOrDefault("E2E_INGRESS_PORT", envOrDefault("INGRESS_PORT", envOrDefault("PORT", "2496")))
	return strings.TrimRight(fmt.Sprintf("https://%s.%s:%s%s", zitiMgmtServiceName, domain, port, zitiMgmtPath), "/")
}

func logZitiResource(t *testing.T, ctx context.Context, session zitiManagementSession, label, path string) {
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
	t.Logf("diagnostics: ziti %s status=%d body=%s", label, response.StatusCode, truncateLogLine(trimmedBody))
}
