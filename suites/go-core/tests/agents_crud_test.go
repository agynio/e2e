//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// The agents service's CRUD surface and its refusals. This lived in
// agents/test/e2e until it was deleted: nothing ran it, so its assertions had
// drifted from the service (it still expected FailedPrecondition where an
// unknown agent now yields NotFound, and covered Hooks, VolumeAttachments and
// ImagePullSecretAttachments, none of which the service implements any more).
// The behaviours below are the ones no other test asserts.

const crudListPageSize int32 = 50

func agentsCRUDContext(t *testing.T) (context.Context, agentsv1.AgentsServiceClient, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	return asOwner(t, ctx), agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr)), gatewayOrganizationID(t)
}

func requireCode(t *testing.T, err error, want codes.Code, what string) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Fatalf("%s: expected %s, got %s (%v)", what, want, got, err)
	}
}

// Every refusal below is a validation the handler performs before it touches
// the store, and each is the only assertion of it anywhere.
func TestAgentsServiceRefusals(t *testing.T) {
	ctx, client, orgID := agentsCRUDContext(t)

	_, err := client.CreateAgent(ctx, &agentsv1.CreateAgentRequest{
		OrganizationId: orgID,
		Name:           "e2e-refusal-" + uuid.NewString()[:8],
		Model:          gatewayModelID(t),
		InitImage:      "",
		Availability:   agentsv1.AgentAvailability_AGENT_AVAILABILITY_INTERNAL,
	})
	requireCode(t, err, codes.InvalidArgument, "CreateAgent without init_image or an environment")

	_, err = client.GetAgent(ctx, &agentsv1.GetAgentRequest{Id: uuid.NewString()})
	requireCode(t, err, codes.NotFound, "GetAgent on an unknown id")

	_, err = client.UpdateAgent(ctx, &agentsv1.UpdateAgentRequest{Id: uuid.NewString()})
	requireCode(t, err, codes.InvalidArgument, "UpdateAgent naming no field")

	// An unknown agent is NotFound, not FailedPrecondition: CreateMcp resolves
	// the organization through the agent before it writes anything.
	_, err = client.CreateMcp(ctx, &agentsv1.CreateMcpRequest{
		AgentId: uuid.NewString(), Name: "e2e_refusal", Command: "mcp",
	})
	requireCode(t, err, codes.NotFound, "CreateMcp naming an unknown agent")

	agent := createAgent(t, ctx, client, "e2e-refusal-"+uuid.NewString(), gatewayModelID(t), orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, ctx, client, agentID) })

	_, err = client.UpdateAgent(ctx, &agentsv1.UpdateAgentRequest{Id: agentID, InitImage: proto.String("")})
	requireCode(t, err, codes.InvalidArgument, "UpdateAgent clearing init_image")

	mcp := createMCP(t, ctx, client, agentID, "e2e_refusal", "node:22-slim", "mcp")
	mcpID := mcp.GetMeta().GetId()
	t.Cleanup(func() { deleteMCP(t, ctx, client, mcpID) })

	// The MCP is a child row, so the agent cannot go while it is there.
	_, err = client.DeleteAgent(ctx, &agentsv1.DeleteAgentRequest{Id: agentID})
	requireCode(t, err, codes.FailedPrecondition, "DeleteAgent while an MCP references it")

	env := createAgentEnv(t, ctx, client, agentID, "E2E_REFUSAL", "value")
	envID := env.GetMeta().GetId()
	t.Cleanup(func() { deleteAgentEnv(t, ctx, client, envID) })

	_, err = client.UpdateEnv(ctx, &agentsv1.UpdateEnvRequest{
		Id: envID, Value: proto.String("value"), SecretId: proto.String(uuid.NewString()),
	})
	requireCode(t, err, codes.InvalidArgument, "UpdateEnv naming both a value and a secret")
}

// A volume's ttl is what decides when an owner's disk is reclaimed, and nothing
// else asserts that it survives a write, an update and a read.
func TestVolumeTTLRoundTrip(t *testing.T) {
	ctx, client, orgID := agentsCRUDContext(t)

	environmentID := createEnvironment(t, ctx, client, &agentsv1.CreateEnvironmentRequest{
		OrganizationId: orgID,
		Name:           "e2e-volume-ttl-" + uuid.NewString()[:8],
		RunnerId:       catalogRunnerID(t, ctx),
		Availability:   agentsv1.EnvironmentAvailability_ENVIRONMENT_AVAILABILITY_INTERNAL,
		Image:          "ghcr.io/agynio/environment:latest",
		Flavor:         "small",
	}).GetMeta().GetId()

	created, err := client.CreateVolume(ctx, &agentsv1.CreateVolumeRequest{
		Target:     &agentsv1.CreateVolumeRequest_EnvironmentId{EnvironmentId: environmentID},
		Name:       "data",
		MountPath:  "/data",
		Size:       "1Gi",
		Persistent: true,
		Ttl:        proto.String("168h"),
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	volumeID := created.GetVolume().GetMeta().GetId()
	if got := created.GetVolume().GetTtl(); got != "168h" {
		t.Fatalf("create volume: expected ttl 168h, got %q", got)
	}

	updated, err := client.UpdateVolume(ctx, &agentsv1.UpdateVolumeRequest{Id: volumeID, Ttl: proto.String("24h")})
	if err != nil {
		t.Fatalf("update volume: %v", err)
	}
	if got := updated.GetVolume().GetTtl(); got != "24h" {
		t.Fatalf("update volume: expected ttl 24h, got %q", got)
	}

	fetched, err := client.GetVolume(ctx, &agentsv1.GetVolumeRequest{Id: volumeID})
	if err != nil {
		t.Fatalf("get volume: %v", err)
	}
	if got := fetched.GetVolume().GetTtl(); got != "24h" {
		t.Fatalf("get volume: expected the persisted ttl 24h, got %q", got)
	}

	if _, err := client.DeleteVolume(ctx, &agentsv1.DeleteVolumeRequest{Id: volumeID}); err != nil {
		t.Fatalf("delete volume: %v", err)
	}
	if _, err := client.GetVolume(ctx, &agentsv1.GetVolumeRequest{Id: volumeID}); status.Code(err) != codes.NotFound {
		t.Fatalf("expected the deleted volume to be gone, got %v", err)
	}
}

// An init script and an ENV both accept an MCP as their target, which resolves
// its organization through the MCP rather than through an agent.
func TestSubResourcesTargetAnMcp(t *testing.T) {
	ctx, client, orgID := agentsCRUDContext(t)

	agent := createAgent(t, ctx, client, "e2e-mcp-target-"+uuid.NewString(), gatewayModelID(t), orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, ctx, client, agentID) })

	mcp := createMCP(t, ctx, client, agentID, "e2e_mcp_target", "node:22-slim", "mcp")
	mcpID := mcp.GetMeta().GetId()
	t.Cleanup(func() { deleteMCP(t, ctx, client, mcpID) })

	script, err := client.CreateInitScript(ctx, &agentsv1.CreateInitScriptRequest{
		Target: &agentsv1.CreateInitScriptRequest_McpId{McpId: mcpID},
		Script: "echo mcp",
	})
	if err != nil {
		t.Fatalf("create mcp init script: %v", err)
	}
	scriptID := script.GetInitScript().GetMeta().GetId()

	// A partial update leaves the field it does not name alone.
	updated, err := client.UpdateInitScript(ctx, &agentsv1.UpdateInitScriptRequest{
		Id: scriptID, Description: proto.String("described"),
	})
	if err != nil {
		t.Fatalf("update mcp init script: %v", err)
	}
	if got := updated.GetInitScript().GetScript(); got != "echo mcp" {
		t.Fatalf("partial update overwrote the script: %q", got)
	}
	if got := updated.GetInitScript().GetDescription(); got != "described" {
		t.Fatalf("partial update did not apply the description: %q", got)
	}

	agentScript, err := client.CreateInitScript(ctx, &agentsv1.CreateInitScriptRequest{
		Target: &agentsv1.CreateInitScriptRequest_AgentId{AgentId: agentID},
		Script: "echo agent",
	})
	if err != nil {
		t.Fatalf("create agent init script: %v", err)
	}

	byAgent, err := client.ListInitScripts(ctx, &agentsv1.ListInitScriptsRequest{AgentId: agentID, PageSize: crudListPageSize})
	if err != nil {
		t.Fatalf("list init scripts by agent: %v", err)
	}
	requireInitScript(t, byAgent.GetInitScripts(), agentScript.GetInitScript().GetMeta().GetId(), "agent filter")
	requireNoInitScript(t, byAgent.GetInitScripts(), scriptID, "agent filter")

	byMcp, err := client.ListInitScripts(ctx, &agentsv1.ListInitScriptsRequest{McpId: mcpID, PageSize: crudListPageSize})
	if err != nil {
		t.Fatalf("list init scripts by mcp: %v", err)
	}
	requireInitScript(t, byMcp.GetInitScripts(), scriptID, "mcp filter")
	requireNoInitScript(t, byMcp.GetInitScripts(), agentScript.GetInitScript().GetMeta().GetId(), "mcp filter")

	mcpEnv := createMCPEnv(t, ctx, client, mcpID, "E2E_MCP_ENV", "value")
	byMcpEnv, err := client.ListEnvs(ctx, &agentsv1.ListEnvsRequest{McpId: mcpID, OrganizationId: orgID, PageSize: crudListPageSize})
	if err != nil {
		t.Fatalf("list envs by mcp: %v", err)
	}
	found := false
	for _, env := range byMcpEnv.GetEnvs() {
		if env.GetMeta().GetId() == mcpEnv.GetMeta().GetId() {
			found = true
		}
	}
	if !found {
		t.Fatalf("mcp filter did not return the mcp's env")
	}
}

// An ENV's source is a oneof: setting a secret has to clear the value it
// replaces, or the row reports two sources.
func TestEnvSourceSwitchesToASecret(t *testing.T) {
	ctx, client, orgID := agentsCRUDContext(t)

	agent := createAgent(t, ctx, client, "e2e-env-source-"+uuid.NewString(), gatewayModelID(t), orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, ctx, client, agentID) })

	env := createAgentEnv(t, ctx, client, agentID, "E2E_ENV_SOURCE", "literal")
	envID := env.GetMeta().GetId()
	t.Cleanup(func() { deleteAgentEnv(t, ctx, client, envID) })

	secretID := uuid.NewString()
	updated, err := client.UpdateEnv(ctx, &agentsv1.UpdateEnvRequest{Id: envID, SecretId: proto.String(secretID)})
	if err != nil {
		t.Fatalf("update env to a secret source: %v", err)
	}
	if got := updated.GetEnv().GetSecretId(); got != secretID {
		t.Fatalf("expected secret_id %s, got %q", secretID, got)
	}
	if got := updated.GetEnv().GetValue(); got != "" {
		t.Fatalf("expected the literal value to be cleared, got %q", got)
	}
}

func requireInitScript(t *testing.T, scripts []*agentsv1.InitScript, id, what string) {
	t.Helper()
	for _, script := range scripts {
		if script.GetMeta().GetId() == id {
			return
		}
	}
	t.Fatalf("%s: expected init script %s in the listing of %d", what, id, len(scripts))
}

func requireNoInitScript(t *testing.T, scripts []*agentsv1.InitScript, id, what string) {
	t.Helper()
	for _, script := range scripts {
		if script.GetMeta().GetId() == id {
			t.Fatalf("%s: init script %s leaked into a listing narrowed to another target", what, id)
		}
	}
}
