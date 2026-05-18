//go:build e2e && (svc_llm || svc_llm_proxy)

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	authorizationv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/authorization/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	llmProxyURLDefault      = "https://llm.agyn.dev:2496"
	llmProxyRequestTimeout  = 45 * time.Second
	llmGatewayServicePath   = "agynio.api.gateway.v1.LLMGateway"
	llmProviderTestEndpoint = "https://testllm.dev/v1/org/agynio/suite/agn/responses"
	llmProxyIdentityType    = "user"
	llmProxyAuthzAddr       = "authorization:50051"
	llmProxyIdentityKey     = "x-identity-id"
	llmProxyIdentityTypeKey = "x-identity-type"
	llmProxyModelRelation   = "org"
	llmProxyModelPrefix     = "model:"
	llmProxyOrgPrefix       = "organization:"
)

func TestLLMProxyURLDefaultUsesIngress(t *testing.T) {
	t.Setenv("E2E_DOMAIN", "e2e.agyn.dev")
	t.Setenv("E2E_INGRESS_PORT", "30443")

	require.Equal(t, "https://llm.e2e.agyn.dev:30443", llmProxyBaseURL())
}

func TestLLMProxyURLExplicitOverride(t *testing.T) {
	t.Setenv("LLM_PROXY_URL", "https://custom.example.test:9443")

	require.Equal(t, "https://custom.example.test:9443", llmProxyBaseURL())
}

func TestLLMGatewayConnectEndpointPath(t *testing.T) {
	t.Setenv(gatewayBaseURLEnvKey, "http://gateway-gateway.platform.svc.cluster.local:8080")

	require.Equal(
		t,
		"http://gateway-gateway.platform.svc.cluster.local:8080/agynio.api.gateway.v1.LLMGateway/CreateLLMProvider",
		gatewayLLMConnectEndpoint(t, "CreateLLMProvider"),
	)
}

func TestLLMProxyGatewayCreatedModel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	usersConn := dialGRPC(t, usersAddr)
	orgsConn := dialGRPC(t, orgsAddr)
	usersClient := usersv1.NewUsersServiceClient(usersConn)
	orgsClient := organizationsv1.NewOrganizationsServiceClient(orgsConn)
	authzClient := newLLMProxyAuthorizationClient(t)

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	apiToken := createAPIToken(t, ctx, usersClient, identityID)
	organizationID := createTestOrganization(t, ctx, orgsClient, identityID)

	providerID, err := createGatewayLLMProvider(t, ctx, apiToken, organizationID)
	if err != nil {
		t.Fatalf("create llm provider through gateway: %v", err)
	}
	modelID, err := createGatewayLLMModel(t, ctx, apiToken, organizationID, providerID)
	if err != nil {
		t.Fatalf("create llm model through gateway: %v", err)
	}
	ensureLLMProxyModelAccess(t, ctx, authzClient, identityID, modelID, organizationID)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), llmProxyRequestTimeout)
		defer cleanupCancel()
		deleteLLMProxyModelAccess(t, cleanupCtx, authzClient, identityID, modelID, organizationID)
		postGatewayConnectBestEffort(t, cleanupCtx, apiToken, "DeleteModel", map[string]string{"id": modelID})
		postGatewayConnectBestEffort(t, cleanupCtx, apiToken, "DeleteLLMProvider", map[string]string{"id": providerID})
		_, _ = orgsClient.DeleteOrganization(withIdentity(cleanupCtx, identityID), &organizationsv1.DeleteOrganizationRequest{Id: organizationID})
	})

	requestBody := llmProxyResponsesRequest{Model: modelID, Input: "hi"}
	responseBody, statusCode := postLLMProxyResponses(t, apiToken, requestBody)

	require.NotEqual(t, http.StatusForbidden, statusCode, "llm-proxy unexpectedly rejected model access: %s", responseBody)
	require.Equal(t, http.StatusOK, statusCode, "llm-proxy request failed: %s", responseBody)
	require.Contains(t, responseBody, "Hi! How are you?")
}

func ensureLLMProxyModelAccess(t *testing.T, ctx context.Context, client authorizationv1.AuthorizationServiceClient, identityID string, modelID string, organizationID string) {
	t.Helper()
	tuple := llmProxyModelAccessTuple(modelID, organizationID)
	_, err := client.Write(llmProxyAdminContext(ctx), &authorizationv1.WriteRequest{Writes: []*authorizationv1.TupleKey{tuple}})
	if err == nil {
		return
	}
	statusErr, ok := status.FromError(err)
	if ok && statusErr.Code() == codes.InvalidArgument && strings.Contains(statusErr.Message(), "already exists") {
		return
	}
	t.Fatalf("authorization write failed: %v", err)
}

func deleteLLMProxyModelAccess(t *testing.T, ctx context.Context, client authorizationv1.AuthorizationServiceClient, identityID string, modelID string, organizationID string) {
	t.Helper()
	tuple := llmProxyModelAccessTuple(modelID, organizationID)
	_, err := client.Write(llmProxyAdminContext(ctx), &authorizationv1.WriteRequest{Deletes: []*authorizationv1.TupleKey{tuple}})
	if err != nil {
		t.Logf("cleanup: authorization delete failed for identity %s model %s: %v", identityID, modelID, err)
	}
}

func llmProxyModelAccessTuple(modelID string, organizationID string) *authorizationv1.TupleKey {
	return &authorizationv1.TupleKey{
		User:     llmProxyOrgPrefix + organizationID,
		Relation: llmProxyModelRelation,
		Object:   llmProxyModelPrefix + modelID,
	}
}

func newLLMProxyAuthorizationClient(t *testing.T) authorizationv1.AuthorizationServiceClient {
	t.Helper()
	conn := dialGRPC(t, envOrDefault("AUTHORIZATION_ADDRESS", llmProxyAuthzAddr))
	return authorizationv1.NewAuthorizationServiceClient(conn)
}

func llmProxyAdminContext(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		llmProxyIdentityKey, clusterAdminIdentityID,
		llmProxyIdentityTypeKey, llmProxyIdentityType,
	))
}

func createGatewayLLMProvider(t *testing.T, ctx context.Context, token string, organizationID string) (string, error) {
	t.Helper()
	resp, err := postGatewayConnect[gatewayCreateLLMProviderResponse](t, ctx, token, "CreateLLMProvider", map[string]string{
		"endpoint":       llmProviderTestEndpoint,
		"authMethod":     "AUTH_METHOD_X_API_KEY",
		"token":          "e2e-test-token",
		"organizationId": organizationID,
		"protocol":       "PROTOCOL_RESPONSES",
	})
	if err != nil {
		return "", err
	}
	providerID := strings.TrimSpace(resp.Provider.Meta.ID)
	if providerID == "" {
		return "", fmt.Errorf("provider id missing")
	}
	return providerID, nil
}

func createGatewayLLMModel(t *testing.T, ctx context.Context, token string, organizationID string, providerID string) (string, error) {
	t.Helper()
	resp, err := postGatewayConnect[gatewayCreateModelResponse](t, ctx, token, "CreateModel", map[string]string{
		"name":           fmt.Sprintf("e2e-llm-proxy-%s", uuid.NewString()),
		"llmProviderId":  providerID,
		"remoteName":     "simple-hello",
		"organizationId": organizationID,
	})
	if err != nil {
		return "", err
	}
	modelID := strings.TrimSpace(resp.Model.Meta.ID)
	if modelID == "" {
		return "", fmt.Errorf("model id missing")
	}
	return modelID, nil
}

type gatewayCreateLLMProviderResponse struct {
	Provider gatewayLLMEntity `json:"provider"`
}

type gatewayCreateModelResponse struct {
	Model gatewayLLMEntity `json:"model"`
}

type gatewayLLMEntity struct {
	Meta gatewayEntityMeta `json:"meta"`
}

type gatewayEntityMeta struct {
	ID string `json:"id"`
}

func postGatewayConnect[T any](t *testing.T, ctx context.Context, token string, method string, payload any) (T, error) {
	t.Helper()
	var response T
	body, statusCode := postGatewayConnectRaw(t, ctx, token, method, payload)
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return response, fmt.Errorf("gateway %s failed with status %d: %s", method, statusCode, body)
	}

	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return response, fmt.Errorf("decode gateway %s response: %w", method, err)
	}
	return response, nil
}

func postGatewayConnectBestEffort(t *testing.T, ctx context.Context, token string, method string, payload any) {
	t.Helper()
	_, _ = postGatewayConnectRaw(t, ctx, token, method, payload)
}

func postGatewayConnectRaw(t *testing.T, ctx context.Context, token string, method string, payload any) (string, int) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal gateway %s request: %v", method, err)
	}

	endpoint := gatewayLLMConnectEndpoint(t, method)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build gateway %s request: %v", method, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connect-Protocol-Version", "1")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := newGatewayClient(t).Do(request)
	if err != nil {
		t.Fatalf("post gateway %s: %v", method, err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read gateway %s response: %v", method, err)
	}
	return strings.TrimSpace(string(responseBody)), response.StatusCode
}

func gatewayLLMConnectEndpoint(t *testing.T, method string) string {
	t.Helper()
	endpoint, err := url.JoinPath(gatewayBaseURL(t), llmGatewayServicePath, method)
	if err != nil {
		t.Fatalf("build gateway %s url: %v", method, err)
	}
	return endpoint
}

type llmProxyResponsesRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

func postLLMProxyResponses(t *testing.T, token string, payload llmProxyResponsesRequest) (string, int) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal llm-proxy request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), llmProxyRequestTimeout)
	defer cancel()

	endpoint := llmProxyBaseURL() + "/v1/responses"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build llm-proxy request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := newGatewayClient(t).Do(request)
	if err != nil {
		t.Fatalf("post llm-proxy request: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read llm-proxy response: %v", err)
	}
	return strings.TrimSpace(string(responseBody)), response.StatusCode
}

func llmProxyBaseURL() string {
	if explicitURL := strings.TrimSpace(envOrDefault("LLM_PROXY_URL", "")); explicitURL != "" {
		return strings.TrimRight(explicitURL, "/")
	}
	domain := envOrDefault("E2E_DOMAIN", envOrDefault("DOMAIN", "agyn.dev"))
	port := envOrDefault("E2E_INGRESS_PORT", envOrDefault("INGRESS_PORT", envOrDefault("PORT", "2496")))
	if strings.TrimSpace(domain) == "agyn.dev" && strings.TrimSpace(port) == "2496" {
		return llmProxyURLDefault
	}
	return strings.TrimRight(fmt.Sprintf("https://llm.%s:%s", domain, port), "/")
}
