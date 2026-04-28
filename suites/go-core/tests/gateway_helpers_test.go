//go:build e2e && (svc_gateway || svc_reminders || smoke)

package tests

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const (
	gatewayBaseURLEnvKey        = "AGYN_BASE_URL"
	gatewayAPITokenEnvKey       = "AGYN_API_TOKEN"
	gatewayOrganizationEnvKey   = "AGYN_ORGANIZATION_ID"
	gatewayModelEnvKey          = "AGYN_MODEL_ID"
	gatewayAgentImageEnvKey     = "AGYN_AGENT_IMAGE"
	gatewayAgentInitImageEnvKey = "AGYN_AGENT_INIT_IMAGE"
	gatewayUsersAddrEnvKey      = "USERS_ADDRESS"
	defaultGatewayUsersAddr     = "users:50051"
	zitiGatewayBaseURL          = "http://gateway"
	gatewayRequestTimeout       = 30 * time.Second
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

func gatewayAgentImage(t *testing.T) string {
	t.Helper()
	return requireGatewayEnv(t, gatewayAgentImageEnvKey)
}

func gatewayAgentInitImage(t *testing.T) string {
	t.Helper()
	return requireGatewayEnv(t, gatewayAgentInitImageEnvKey)
}

func gatewayUsersAddr() string {
	return envOrDefault(gatewayUsersAddrEnvKey, defaultGatewayUsersAddr)
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

func gatewayUserToken(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	usersConn := dialGRPC(t, gatewayUsersAddr())
	usersClient := usersv1.NewUsersServiceClient(usersConn)

	subject := fmt.Sprintf("e2e-gateway-%s", uuid.NewString())
	name := fmt.Sprintf("E2E Gateway %s", subject)
	email := fmt.Sprintf("%s@test.local", subject)

	resp, err := usersClient.ResolveOrCreateUser(ctx, &usersv1.ResolveOrCreateUserRequest{
		OidcSubject: subject,
		Name:        name,
		Email:       email,
	})
	require.NoError(t, err)
	if resp == nil || resp.GetUser() == nil || resp.GetUser().GetMeta() == nil {
		t.Fatal("resolve user: missing user metadata")
	}
	identityID := strings.TrimSpace(resp.GetUser().GetMeta().GetId())
	if identityID == "" {
		t.Fatal("resolve user: identity id missing")
	}

	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("x-identity-id", identityID))
	tokenResp, err := usersClient.CreateAPIToken(callCtx, &usersv1.CreateAPITokenRequest{
		Name: fmt.Sprintf("e2e-gateway-token-%s", uuid.NewString()),
	})
	require.NoError(t, err)
	if tokenResp == nil || tokenResp.GetToken() == nil {
		t.Fatal("create api token: missing response")
	}
	plaintext := strings.TrimSpace(tokenResp.GetPlaintextToken())
	if plaintext == "" {
		t.Fatal("create api token: plaintext token missing")
	}
	return plaintext
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
