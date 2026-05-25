//go:build e2e && (svc_agents_orchestrator || svc_llm || svc_llm_proxy || smoke)

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

	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	"github.com/google/uuid"
)

const (
	llmGatewayServicePath   = "agynio.api.gateway.v1.LLMGateway"
	llmProviderTestEndpoint = "https://testllm.dev/v1/org/agynio/suite/agn/responses"
)

func createGatewayLLMProvider(t *testing.T, ctx context.Context, token string, organizationID string) (string, error) {
	t.Helper()
	provider, err := createGatewayLLMProviderResource(
		t,
		ctx,
		token,
		llmProviderTestEndpoint,
		"AUTH_METHOD_X_API_KEY",
		"e2e-test-token",
		organizationID,
		llmv1.Protocol_PROTOCOL_RESPONSES,
	)
	if err != nil {
		return "", err
	}
	return provider.GetMeta().GetId(), nil
}

func createGatewayLLMProviderResource(
	t *testing.T,
	ctx context.Context,
	token string,
	endpoint string,
	authMethod string,
	providerToken string,
	organizationID string,
	protocol llmv1.Protocol,
) (*llmv1.LLMProvider, error) {
	t.Helper()
	resp, err := postGatewayConnect[gatewayCreateLLMProviderResponse](t, ctx, token, "CreateLLMProvider", map[string]string{
		"endpoint":       endpoint,
		"authMethod":     authMethod,
		"token":          providerToken,
		"organizationId": organizationID,
		"protocol":       protocol.String(),
	})
	if err != nil {
		return nil, err
	}
	providerID := strings.TrimSpace(resp.Provider.Meta.ID)
	if providerID == "" {
		return nil, fmt.Errorf("provider id missing")
	}
	return &llmv1.LLMProvider{Meta: &llmv1.EntityMeta{Id: providerID}}, nil
}

func createGatewayLLMModel(t *testing.T, ctx context.Context, token string, organizationID string, providerID string) (string, error) {
	t.Helper()
	model, err := createGatewayLLMModelResource(
		t,
		ctx,
		token,
		fmt.Sprintf("e2e-llm-proxy-%s", uuid.NewString()),
		organizationID,
		providerID,
		"simple-hello",
	)
	if err != nil {
		return "", err
	}
	return model.GetMeta().GetId(), nil
}

func createGatewayLLMModelResource(
	t *testing.T,
	ctx context.Context,
	token string,
	name string,
	organizationID string,
	providerID string,
	remoteName string,
) (*llmv1.Model, error) {
	t.Helper()
	resp, err := postGatewayConnect[gatewayCreateModelResponse](t, ctx, token, "CreateModel", map[string]string{
		"name":           name,
		"llmProviderId":  providerID,
		"remoteName":     remoteName,
		"organizationId": organizationID,
	})
	if err != nil {
		return nil, err
	}
	modelID := strings.TrimSpace(resp.Model.Meta.ID)
	if modelID == "" {
		return nil, fmt.Errorf("model id missing")
	}
	return &llmv1.Model{Meta: &llmv1.EntityMeta{Id: modelID}}, nil
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
