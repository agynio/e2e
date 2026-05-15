//go:build e2e && (svc_llm || svc_llm_proxy)

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	llmv1connect "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1/llmv1connect"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	llmProxyURLDefault      = "http://llm-proxy:8080"
	llmProxyRequestTimeout  = 45 * time.Second
	llmProviderTestEndpoint = "https://testllm.dev/v1/org/agynio/suite/agn/responses"
)

func TestLLMProxyGatewayCreatedModel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	usersConn := dialGRPC(t, usersAddr)
	orgsConn := dialGRPC(t, orgsAddr)
	usersClient := usersv1.NewUsersServiceClient(usersConn)
	orgsClient := organizationsv1.NewOrganizationsServiceClient(orgsConn)

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	apiToken := createAPIToken(t, ctx, usersClient, identityID)
	organizationID := createTestOrganization(t, ctx, orgsClient, identityID)

	llmClient := newLLMGatewayClient(t, apiToken)
	providerID := createGatewayLLMProvider(t, ctx, llmClient, organizationID)
	modelID := createGatewayLLMModel(t, ctx, llmClient, organizationID, providerID)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), llmProxyRequestTimeout)
		defer cleanupCancel()
		_, _ = llmClient.DeleteModel(cleanupCtx, connect.NewRequest(&llmv1.DeleteModelRequest{Id: modelID}))
		_, _ = llmClient.DeleteLLMProvider(cleanupCtx, connect.NewRequest(&llmv1.DeleteLLMProviderRequest{Id: providerID}))
		_, _ = orgsClient.DeleteOrganization(withIdentity(cleanupCtx, identityID), &organizationsv1.DeleteOrganizationRequest{Id: organizationID})
	})

	requestBody := llmProxyResponsesRequest{Model: modelID, Input: "hi"}
	responseBody, statusCode := postLLMProxyResponses(t, apiToken, requestBody)

	require.NotEqual(t, http.StatusForbidden, statusCode, "llm-proxy unexpectedly rejected model access: %s", responseBody)
	require.Equal(t, http.StatusOK, statusCode, "llm-proxy request failed: %s", responseBody)
	require.Contains(t, responseBody, "Hi! How are you?")
}

func newLLMGatewayClient(t *testing.T, token string) llmv1connect.LLMServiceClient {
	t.Helper()
	return llmv1connect.NewLLMServiceClient(
		newGatewayAuthenticatedClient(t, token),
		gatewayBaseURL(t),
	)
}

func createGatewayLLMProvider(t *testing.T, ctx context.Context, client llmv1connect.LLMServiceClient, organizationID string) string {
	t.Helper()
	resp, err := client.CreateLLMProvider(ctx, connect.NewRequest(&llmv1.CreateLLMProviderRequest{
		Endpoint:       llmProviderTestEndpoint,
		AuthMethod:     llmv1.AuthMethod_AUTH_METHOD_X_API_KEY,
		Token:          "e2e-test-token",
		OrganizationId: organizationID,
		Protocol:       llmv1.Protocol_PROTOCOL_RESPONSES,
	}))
	require.NoError(t, err)
	providerID := strings.TrimSpace(resp.Msg.GetProvider().GetMeta().GetId())
	require.NotEmpty(t, providerID)
	return providerID
}

func createGatewayLLMModel(t *testing.T, ctx context.Context, client llmv1connect.LLMServiceClient, organizationID string, providerID string) string {
	t.Helper()
	resp, err := client.CreateModel(ctx, connect.NewRequest(&llmv1.CreateModelRequest{
		Name:           fmt.Sprintf("e2e-llm-proxy-%s", uuid.NewString()),
		LlmProviderId:  providerID,
		RemoteName:     "simple-hello",
		OrganizationId: organizationID,
	}))
	require.NoError(t, err)
	modelID := strings.TrimSpace(resp.Msg.GetModel().GetMeta().GetId())
	require.NotEmpty(t, modelID)
	return modelID
}

type llmProxyResponsesRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

func postLLMProxyResponses(t *testing.T, token string, payload llmProxyResponsesRequest) (string, int) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), llmProxyRequestTimeout)
	defer cancel()

	endpoint := strings.TrimRight(envOrDefault("LLM_PROXY_URL", llmProxyURLDefault), "/") + "/v1/responses"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := newGatewayClient(t).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return strings.TrimSpace(string(responseBody)), response.StatusCode
}
