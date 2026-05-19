//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
)

const (
	llmProxyAgentThreadTestName = "e2e-llm-proxy-agent-thread"
	llmProxyAgentThreadMessage  = "hi"
	llmProxyAgentThreadResponse = "Hi! How are you?"
)

func TestAgentRespondsToThreadMessageViaLLMProxy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	runnerConn := dialRunnerGRPC(t, runnerAddr)
	usersConn := dialGRPC(t, usersAddr)
	orgsConn := dialGRPC(t, orgsAddr)
	llmConn := dialGRPC(t, llmAddr)

	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)
	runnerClient := runnerv1.NewRunnerServiceClient(runnerConn)
	usersClient := usersv1.NewUsersServiceClient(usersConn)
	orgsClient := organizationsv1.NewOrganizationsServiceClient(orgsConn)
	llmClient := llmv1.NewLLMServiceClient(llmConn)

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	threadsCtx := withIdentity(ctx, identityID)
	apiToken := createAPIToken(t, ctx, usersClient, identityID)
	organizationID := createTestOrganization(t, ctx, orgsClient, identityID)

	provider := createLLMProviderWithProtocol(t, ctx, llmClient, llmProviderTestEndpoint, organizationID, llmv1.Protocol_PROTOCOL_RESPONSES)
	providerID := provider.GetMeta().GetId()
	if providerID == "" {
		t.Fatal("create llm provider: missing id")
	}
	model := createModel(t, ctx, llmClient, "e2e-llm-proxy-model-"+uuid.NewString(), providerID, "simple-hello", organizationID)
	modelID := model.GetMeta().GetId()
	if modelID == "" {
		t.Fatal("create model: missing id")
	}

	agent := createAgent(t, ctx, agentsClient, fmt.Sprintf("%s-%s", llmProxyAgentThreadTestName, uuid.NewString()), modelID, organizationID, agnInitImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() { deleteAgent(t, ctx, agentsClient, agentID) })
	createAgentEnv(t, ctx, agentsClient, agentID, "LLM_API_TOKEN", apiToken)

	thread := createThread(t, threadsCtx, threadsClient, organizationID, []string{identityID, agentID})
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

	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadID, identityID, llmProxyAgentThreadMessage)
	sentMessageTime := messageCreatedAt(t, sentMessage)
	startTimeMinNs := messageStartTimeMinNs(t, sentMessage)

	pollCtx, pollCancel := context.WithTimeout(threadsCtx, 5*time.Minute)
	defer pollCancel()
	agentBody, err := pollForAgentResponse(t, pollCtx, threadsClient, runnerClient, threadID, agentID, labels, sentMessageTime, llmProxyAgentThreadResponse)
	if err != nil {
		logShellToolExecutionDiagnostics(t, startTimeMinNs, organizationID, threadID)
		t.Fatalf("wait for agent response: %v", err)
	}
	if agentBody != llmProxyAgentThreadResponse {
		t.Fatalf("expected agent response %q, got %q", llmProxyAgentThreadResponse, agentBody)
	}
}
