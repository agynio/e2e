//go:build e2e && svc_gateway

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	gatewayv1connect "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/gateway/v1/gatewayv1connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAgentsGateway_ListAgents(t *testing.T) {
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

func TestAgentsGateway_CreateAndDeleteAgent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newAgentsGatewayClient(t)
	createReq := &agentsv1.CreateAgentRequest{
		Name:           fmt.Sprintf("e2e-gateway-agent-%s", uuid.NewString()),
		Role:           "assistant",
		Model:          gatewayModelID(t),
		Configuration:  "{}",
		Image:          gatewayAgentImage(t),
		InitImage:      gatewayAgentInitImage(t),
		OrganizationId: gatewayOrganizationID(t),
	}
	createResp, err := client.CreateAgent(ctx, connect.NewRequest(createReq))
	require.NoError(t, err)
	require.NotNil(t, createResp)
	require.NotNil(t, createResp.Msg)
	require.NotNil(t, createResp.Msg.Agent)
	require.NotNil(t, createResp.Msg.Agent.Meta)

	agentID := strings.TrimSpace(createResp.Msg.Agent.Meta.Id)
	require.NotEmpty(t, agentID)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
		defer cleanupCancel()
		_, _ = client.DeleteAgent(cleanupCtx, connect.NewRequest(&agentsv1.DeleteAgentRequest{Id: agentID}))
	})

	_, err = client.DeleteAgent(ctx, connect.NewRequest(&agentsv1.DeleteAgentRequest{Id: agentID}))
	require.NoError(t, err)
}

func TestAgentsGateway_ListMcps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newAgentsGatewayClient(t)
	agentID := createGatewayAgent(t, client)
	resp, err := client.ListMcps(ctx, connect.NewRequest(&agentsv1.ListMcpsRequest{AgentId: agentID}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Msg)
}

func TestAgentsGateway_InvalidPayloadReturnsClientError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	client := newAgentsGatewayClient(t)
	_, err := client.GetAgent(ctx, connect.NewRequest(&agentsv1.GetAgentRequest{}))
	require.Error(t, err)
}

func newAgentsGatewayClient(t *testing.T) gatewayv1connect.AgentsGatewayClient {
	t.Helper()
	return gatewayv1connect.NewAgentsGatewayClient(
		newGatewayAuthenticatedClient(t, gatewayAPIToken(t)),
		gatewayBaseURL(t),
	)
}

func createGatewayAgent(t *testing.T, client gatewayv1connect.AgentsGatewayClient) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	resp, err := client.CreateAgent(ctx, connect.NewRequest(&agentsv1.CreateAgentRequest{
		Name:           fmt.Sprintf("e2e-gateway-agent-%s", uuid.NewString()),
		Role:           "assistant",
		Model:          gatewayModelID(t),
		Configuration:  "{}",
		Image:          gatewayAgentImage(t),
		InitImage:      gatewayAgentInitImage(t),
		OrganizationId: gatewayOrganizationID(t),
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Msg)
	require.NotNil(t, resp.Msg.Agent)
	require.NotNil(t, resp.Msg.Agent.Meta)

	agentID := strings.TrimSpace(resp.Msg.Agent.Meta.Id)
	require.NotEmpty(t, agentID)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
		defer cleanupCancel()
		_, _ = client.DeleteAgent(cleanupCtx, connect.NewRequest(&agentsv1.DeleteAgentRequest{Id: agentID}))
	})

	return agentID
}

func hasGatewayAgentID(agents []*agentsv1.Agent, agentID string) bool {
	for _, agent := range agents {
		if strings.TrimSpace(agent.GetMeta().GetId()) == agentID {
			return true
		}
	}
	return false
}
