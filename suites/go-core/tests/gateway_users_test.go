//go:build e2e && svc_gateway

package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	gatewayv1connect "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/gateway/v1/gatewayv1connect"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const gatewayInvalidAPIToken = "agyn_invalid"

func TestAPIToken_MeEndpoint(t *testing.T) {
	payload := fetchGatewayIdentity(t, gatewayAPIToken(t))
	require.NotEmpty(t, payload.IdentityID)
	require.NotEmpty(t, payload.IdentityType)
}

func TestAPIToken_MeEndpointInvalidToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, gatewayEndpoint(t, "me"), nil)
	require.NoError(t, err)

	response, err := newGatewayAuthenticatedClient(t, gatewayInvalidAPIToken).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.StatusCode)
	}
}

func TestAPIToken_ConnectRPCEndpointAuthenticated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newAgentsGatewayClient(t)
	agentID := createGatewayAgent(t, client)
	resp, err := client.ListAgents(ctx, connect.NewRequest(&agentsv1.ListAgentsRequest{
		OrganizationId: gatewayOrganizationID(t),
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Msg)
	require.NotNil(t, resp.Msg.Agents)
	require.True(t, hasGatewayAgentID(resp.Msg.Agents, agentID))
}

func TestAPIToken_ConnectRPCEndpointInvalidToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := gatewayv1connect.NewAgentsGatewayClient(
		newGatewayAuthenticatedClient(t, gatewayInvalidAPIToken),
		gatewayBaseURL(t),
	)
	_, err := client.ListAgents(ctx, connect.NewRequest(&agentsv1.ListAgentsRequest{
		OrganizationId: gatewayOrganizationID(t),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestUsersGateway_CreateAndRevokeAPIToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newUsersGatewayClient(t)
	createResp, err := client.CreateAPIToken(ctx, connect.NewRequest(&usersv1.CreateAPITokenRequest{
		Name: fmt.Sprintf("e2e-api-token-%d", time.Now().UnixNano()),
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp)
	require.NotNil(t, createResp.Msg)
	require.NotNil(t, createResp.Msg.Token)

	tokenID := strings.TrimSpace(createResp.Msg.Token.Id)
	require.NotEmpty(t, tokenID)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
		defer cleanupCancel()
		_, _ = client.RevokeAPIToken(cleanupCtx, connect.NewRequest(&usersv1.RevokeAPITokenRequest{TokenId: tokenID}))
	})

	listResp, err := client.ListAPITokens(ctx, connect.NewRequest(&usersv1.ListAPITokensRequest{}))
	require.NoError(t, err)
	require.NotNil(t, listResp)
	require.NotNil(t, listResp.Msg)
	require.True(t, hasGatewayTokenID(listResp.Msg.Tokens, tokenID), "expected token to be listed")

	_, err = client.RevokeAPIToken(ctx, connect.NewRequest(&usersv1.RevokeAPITokenRequest{TokenId: tokenID}))
	require.NoError(t, err)

	listResp, err = client.ListAPITokens(ctx, connect.NewRequest(&usersv1.ListAPITokensRequest{}))
	require.NoError(t, err)
	require.NotNil(t, listResp)
	require.NotNil(t, listResp.Msg)
	require.False(t, hasGatewayTokenID(listResp.Msg.Tokens, tokenID), "expected token to be revoked")
}

func TestUsersGateway_ListAPITokens(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newUsersGatewayClient(t)
	resp, err := client.ListAPITokens(ctx, connect.NewRequest(&usersv1.ListAPITokensRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Msg)
	require.NotNil(t, resp.Msg.Tokens)
}

func TestUsersGateway_RevokeAPITokenNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newUsersGatewayClient(t)
	_, err := client.RevokeAPIToken(ctx, connect.NewRequest(&usersv1.RevokeAPITokenRequest{TokenId: uuid.NewString()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestUsersGateway_CreateAPITokenUnauthenticated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := gatewayv1connect.NewUsersGatewayClient(newGatewayClient(t), gatewayBaseURL(t))
	_, err := client.CreateAPIToken(ctx, connect.NewRequest(&usersv1.CreateAPITokenRequest{Name: "e2e-unauth"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAPIToken_CreatedTokenAuthenticates(t *testing.T) {
	baseToken := gatewayUserToken(t)
	baseIdentity := fetchGatewayIdentity(t, baseToken)

	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newUsersGatewayClientWithToken(t, baseToken)
	createResp, err := client.CreateAPIToken(ctx, connect.NewRequest(&usersv1.CreateAPITokenRequest{
		Name: fmt.Sprintf("e2e-roundtrip-token-%d", time.Now().UnixNano()),
	}))
	require.NoError(t, err)
	require.NotNil(t, createResp)
	require.NotNil(t, createResp.Msg)
	require.NotNil(t, createResp.Msg.Token)
	require.NotEmpty(t, createResp.Msg.Token.Id)
	require.NotEmpty(t, createResp.Msg.PlaintextToken)
	require.True(t, strings.HasPrefix(createResp.Msg.PlaintextToken, "agyn_"))

	tokenID := strings.TrimSpace(createResp.Msg.Token.Id)
	plaintextToken := strings.TrimSpace(createResp.Msg.PlaintextToken)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
		defer cleanupCancel()
		_, _ = client.RevokeAPIToken(cleanupCtx, connect.NewRequest(&usersv1.RevokeAPITokenRequest{TokenId: tokenID}))
	})

	newIdentity := fetchGatewayIdentity(t, plaintextToken)
	require.Equal(t, baseIdentity.IdentityID, newIdentity.IdentityID)
}

func newUsersGatewayClient(t *testing.T) gatewayv1connect.UsersGatewayClient {
	t.Helper()
	return newUsersGatewayClientWithToken(t, gatewayUserToken(t))
}

func newUsersGatewayClientWithToken(t *testing.T, token string) gatewayv1connect.UsersGatewayClient {
	t.Helper()
	return gatewayv1connect.NewUsersGatewayClient(
		newGatewayAuthenticatedClient(t, token),
		gatewayBaseURL(t),
	)
}

func hasGatewayTokenID(tokens []*usersv1.APIToken, tokenID string) bool {
	for _, token := range tokens {
		if token.GetId() == tokenID {
			return true
		}
	}
	return false
}
