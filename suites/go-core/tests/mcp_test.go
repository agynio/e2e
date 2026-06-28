//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	"github.com/google/uuid"
)

func TestMCPToolsE2E(t *testing.T) {
	runMCPToolsE2E(t, testLLMEndpointCodex, codexInitImage)
}

func TestMCPToolsAgnE2E(t *testing.T) {
	runMCPToolsE2E(t, testLLMEndpointAgn, agnInitImage)
}

func runMCPToolsE2E(t *testing.T, llmEndpoint, initImage string) pipelineRun {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	runnerConn := dialRunnerGRPC(t, runnerAddr)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)
	runnerClient := runnerv1.NewRunnerServiceClient(runnerConn)

	setup := newWorkflowGatewaySetup(t, ctx)
	identityID := setup.IdentityID
	threadsCtx := setup.Context
	orgID := setup.OrganizationID
	token := setup.Token
	modelID := createWorkflowGatewayModel(t, setup, llmEndpoint, llmv1.Protocol_PROTOCOL_RESPONSES, "mcp-tools-test")

	agent := createAgent(t, threadsCtx, agentsClient, "e2e-mcp-tools-"+uuid.NewString(), modelID, orgID, initImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() {
		cleanupAgentEnvs(t, threadsCtx, agentsClient, agentID)
		deleteAgent(t, threadsCtx, agentsClient, agentID)
	})
	createAgentEnv(t, threadsCtx, agentsClient, agentID, "LLM_API_TOKEN", token)
	memoryMCP := createMCP(
		t,
		ctx,
		agentsClient,
		agentID,
		"memory",
		"node:22-slim",
		`npx -y supergateway@3.4.3 --stdio "npx -y @modelcontextprotocol/server-memory@2026.1.26" --outputTransport streamableHttp --port $MCP_PORT --streamableHttpPath /mcp`,
	)
	memoryMcpID := memoryMCP.GetMeta().GetId()
	if memoryMcpID == "" {
		t.Fatal("create memory mcp: missing id")
	}
	t.Cleanup(func() { deleteMCP(t, ctx, agentsClient, memoryMcpID) })
	createMCPEnv(t, ctx, agentsClient, memoryMcpID, "MEMORY_FILE_PATH", "/tmp/memory.json")

	filesystemMCP := createMCP(
		t,
		ctx,
		agentsClient,
		agentID,
		"filesystem",
		"node:22-slim",
		`mkdir -p /test-data && printf 'hello' > /test-data/hello.txt && npx -y supergateway@3.4.3 --stdio "npx -y @modelcontextprotocol/server-filesystem@2026.1.14 /test-data" --outputTransport streamableHttp --port $MCP_PORT --streamableHttpPath /mcp`,
	)
	filesystemMcpID := filesystemMCP.GetMeta().GetId()
	if filesystemMcpID == "" {
		t.Fatal("create filesystem mcp: missing id")
	}
	t.Cleanup(func() { deleteMCP(t, ctx, agentsClient, filesystemMcpID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{agentID})
	threadID := thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	labels := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentID,
		labelThreadID:  threadID,
	}
	t.Cleanup(func() {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			t.Logf("cleanup: find workloads: %v", err)
			return
		}
		for _, workloadID := range ids {
			cleanupWorkload(t, ctx, runnerClient, workloadID)
		}
	})

	message := "Create an entity called test_project of type project with observation 'A test project', then list files in /test-data"
	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadID, identityID, message)
	sentMessageTime := messageCreatedAt(t, sentMessage)
	startTimeMinNs := messageStartTimeMinNs(t, sentMessage)
	t.Logf("test setup complete: agentID=%s threadID=%s memoryMcpID=%s filesystemMcpID=%s", agentID, threadID, memoryMcpID, filesystemMcpID)

	expected := "I've created the entity 'test_project' (type: project) with the observation 'A test project'. The /test-data directory contains one file: hello.txt."

	readyCtx, readyCancel := context.WithTimeout(ctx, 4*time.Minute)
	defer readyCancel()
	if err := waitForMcpSidecarsReady(t, readyCtx, runnerClient, labels); err != nil {
		t.Fatalf("wait for mcp sidecars: %v", err)
	}

	pollCtx, pollCancel := context.WithTimeout(threadsCtx, 6*time.Minute)
	defer pollCancel()
	agentBody, err := pollForAgentResponse(t, pollCtx, threadsClient, runnerClient, threadID, agentID, labels, sentMessageTime, expected)
	if err != nil {
		t.Fatalf("wait for agent response: %v", err)
	}
	if agentBody != expected {
		t.Fatalf("expected agent response %q, got %q", expected, agentBody)
	}

	return pipelineRun{
		threadID:       threadID,
		organizationID: orgID,
		startTimeMinNs: startTimeMinNs,
		agentResponse:  agentBody,
		messageText:    message,
	}
}
