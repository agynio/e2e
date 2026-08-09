//go:build e2e && (svc_gateway || svc_agents_orchestrator || svc_runners || svc_metering || svc_k8s_runner || svc_organizations || svc_llm || svc_llm_proxy || svc_images || smoke)

package tests

import (
	"context"
	"sync"
	"testing"

	"connectrpc.com/connect"
	gatewayv1connect "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/gateway/v1/gatewayv1connect"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	"github.com/google/uuid"
)

// The model every suite that needs one shares. Resolved once per run rather
// than per test: creating one is a write against the organization, and a test
// that only reads should not depend on the order it ran in.
var (
	suiteModelOnce sync.Once
	suiteModelID   string
	suiteModelErr  error
)

func ensureSuiteModel(t *testing.T) string {
	t.Helper()
	suiteModelOnce.Do(func() {
		suiteModelID, suiteModelErr = resolveOrCreateSuiteModel(t)
	})
	if suiteModelErr != nil {
		t.Fatalf("resolve a model for the suite: %v", suiteModelErr)
	}
	return suiteModelID
}

func resolveOrCreateSuiteModel(t *testing.T) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	token := gatewayAPIToken(t)
	organizationID := gatewayOrganizationID(t)
	identity := fetchGatewayIdentity(t, token)
	ctx = withIdentity(ctx, identity.IdentityID)
	client := gatewayv1connect.NewLLMGatewayClient(newGatewayAuthenticatedClient(t, token), gatewayBaseURL(t))

	existing, err := client.ListModels(ctx, connect.NewRequest(&llmv1.ListModelsRequest{
		OrganizationId: organizationID,
		PageSize:       1,
	}))
	if err != nil {
		return "", err
	}
	if models := existing.Msg.GetModels(); len(models) > 0 {
		return models[0].GetMeta().GetId(), nil
	}

	// The endpoint is a placeholder: the suites need the model row to exist,
	// not to answer completions. Anything that actually calls a provider brings
	// its own.
	provider, err := client.CreateLLMProvider(ctx, connect.NewRequest(&llmv1.CreateLLMProviderRequest{
		Endpoint:       "https://llm.agyn.dev/e2e",
		AuthMethod:     llmv1.AuthMethod_AUTH_METHOD_BEARER,
		Token:          "e2e-token",
		OrganizationId: organizationID,
	}))
	if err != nil {
		return "", err
	}

	model, err := client.CreateModel(ctx, connect.NewRequest(&llmv1.CreateModelRequest{
		Name:           "e2e-suite-model-" + uuid.NewString(),
		LlmProviderId:  provider.Msg.GetProvider().GetMeta().GetId(),
		RemoteName:     "gpt-4o-mini",
		OrganizationId: organizationID,
	}))
	if err != nil {
		return "", err
	}
	return model.Msg.GetModel().GetMeta().GetId(), nil
}
