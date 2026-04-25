//go:build e2e && (svc_gateway || smoke)

package tests

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	gatewayBaseURLEnvKey      = "AGYN_BASE_URL"
	gatewayAPITokenEnvKey     = "AGYN_API_TOKEN"
	gatewayOrganizationEnvKey = "AGYN_ORGANIZATION_ID"
	gatewayModelEnvKey        = "AGYN_MODEL_ID"
	zitiManagementAddrEnvKey  = "ZITI_MANAGEMENT_ADDRESS"
	defaultZitiManagementAddr = "ziti-management:50051"
	zitiGatewayBaseURL        = "http://gateway"
	gatewayRequestTimeout     = 30 * time.Second
)

type gatewayMePayload struct {
	IdentityID   string `json:"identity_id"`
	IdentityType string `json:"identity_type"`
}

func requireGatewayEnv(t *testing.T, key string) string {
	t.Helper()
	value, ok := os.LookupEnv(key)
	if !ok {
		t.Fatalf("%s not set", key)
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		t.Fatalf("%s empty", key)
	}
	return trimmed
}

func gatewayBaseURL(t *testing.T) string {
	t.Helper()
	baseURL := strings.TrimRight(requireGatewayEnv(t, gatewayBaseURLEnvKey), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse %s: %v", gatewayBaseURLEnvKey, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		t.Fatalf("%s must start with http or https", gatewayBaseURLEnvKey)
	}
	if parsed.Host == "" {
		t.Fatalf("%s missing host", gatewayBaseURLEnvKey)
	}
	return baseURL
}

func gatewayAPIToken(t *testing.T) string {
	t.Helper()
	return requireGatewayEnv(t, gatewayAPITokenEnvKey)
}

func gatewayOrganizationID(t *testing.T) string {
	t.Helper()
	return requireGatewayEnv(t, gatewayOrganizationEnvKey)
}

func gatewayModelID(t *testing.T) string {
	t.Helper()
	return requireGatewayEnv(t, gatewayModelEnvKey)
}

func zitiManagementAddr(t *testing.T) string {
	t.Helper()
	return envOrDefault(zitiManagementAddrEnvKey, defaultZitiManagementAddr)
}

func gatewayEndpoint(t *testing.T, path string) string {
	t.Helper()
	endpoint, err := url.JoinPath(gatewayBaseURL(t), path)
	if err != nil {
		t.Fatalf("join gateway path %q: %v", path, err)
	}
	return endpoint
}

func newGatewayClient(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: gatewayTransport(t),
	}
}

func newGatewayAuthenticatedClient(t *testing.T, token string) *http.Client {
	t.Helper()
	client := newGatewayClient(t)
	client.Transport = gatewayBearerTransport{token: token, base: client.Transport}
	return client
}

func gatewayTransport(t *testing.T) http.RoundTripper {
	t.Helper()
	baseURL := gatewayBaseURL(t)
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse %s: %v", gatewayBaseURLEnvKey, err)
	}
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatal("unexpected default transport type")
	}
	transport := baseTransport.Clone()
	if parsed.Scheme == "https" {
		tlsConfig := transport.TLSClientConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		} else {
			tlsConfig = tlsConfig.Clone()
		}
		tlsConfig.InsecureSkipVerify = true
		transport.TLSClientConfig = tlsConfig
	}
	return transport
}

type gatewayBearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t gatewayBearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return base.RoundTrip(clone)
}

func fetchGatewayIdentity(t *testing.T, token string) gatewayMePayload {
	t.Helper()
	payload := fetchGatewayMePayload(t, newGatewayAuthenticatedClient(t, token))
	payload.IdentityID = strings.TrimSpace(payload.IdentityID)
	payload.IdentityType = strings.TrimSpace(payload.IdentityType)
	if payload.IdentityID == "" {
		t.Fatal("me endpoint identity_id missing")
	}
	if payload.IdentityType == "" {
		t.Fatal("me endpoint identity_type missing")
	}
	return payload
}

func fetchGatewayMePayload(t *testing.T, client *http.Client) gatewayMePayload {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, gatewayEndpoint(t, "me"), nil)
	require.NoError(t, err)

	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("me endpoint failed: status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload gatewayMePayload
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	return payload
}
