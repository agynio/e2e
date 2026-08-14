//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	"github.com/google/uuid"
)

func TestFullPipelineMessageResponse(t *testing.T) {
	runFullPipelineMessageResponse(t, testLLMEndpointCodex, codexInitImage, "hello", "Hi! How are you?")
}

func TestFullPipelineAgnMessageResponse(t *testing.T) {
	runFullPipelineMessageResponse(t, testLLMEndpointAgn, agnInitImage, "hi", "Hi! How are you?")
}

func TestFullPipelineClaudeMessageResponse(t *testing.T) {
	runFullPipelineMessageResponseWithProtocol(t, testLLMEndpointClaude, claudeInitImage, llmv1.Protocol_PROTOCOL_ANTHROPIC_MESSAGES, "hello", "Hi! How are you?")
}

func runFullPipelineMessageResponse(t *testing.T, llmEndpoint, initImage, message, expectedResponse string) pipelineRun {
	t.Helper()
	return runFullPipelineMessageResponseWithProtocol(t, llmEndpoint, initImage, llmv1.Protocol_PROTOCOL_RESPONSES, message, expectedResponse)
}

func runFullPipelineMessageResponseWithProtocol(t *testing.T, llmEndpoint, initImage string, protocol llmv1.Protocol, message, expectedResponse string) pipelineRun {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
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
	modelID := createWorkflowGatewayModel(t, setup, llmEndpoint, protocol, "simple-hello")

	agent := createAgent(t, threadsCtx, agentsClient, fmt.Sprintf("e2e-pipeline-%s", uuid.NewString()), modelID, orgID, initImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() {
		cleanupAgentEnvs(t, threadsCtx, agentsClient, agentID)
		deleteAgent(t, threadsCtx, agentsClient, agentID)
	})
	createAgentEnv(t, threadsCtx, agentsClient, agentID, "LLM_API_TOKEN", token)

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID := thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadID, identityID, message)
	sentMessageTime := messageCreatedAt(t, sentMessage)
	startTimeMinNs := messageStartTimeMinNs(t, sentMessage)

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

	pollCtx, pollCancel := context.WithTimeout(threadsCtx, 5*time.Minute)
	defer pollCancel()
	agentBody, err := pollForAgentResponse(t, pollCtx, threadsClient, runnerClient, threadID, agentID, labels, sentMessageTime, expectedResponse)
	if err != nil {
		t.Fatalf("wait for agent response: %v", err)
	}
	if agentBody != expectedResponse {
		t.Fatalf("expected agent response %q, got %q", expectedResponse, agentBody)
	}

	return pipelineRun{
		threadID:       threadID,
		organizationID: orgID,
		startTimeMinNs: startTimeMinNs,
		agentResponse:  agentBody,
		messageText:    message,
	}
}
