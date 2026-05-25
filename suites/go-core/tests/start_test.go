//go:build e2e && (svc_agents_orchestrator || smoke)

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	"github.com/google/uuid"
)

func TestWorkloadStartsOnUnackedMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	runnerConn := dialRunnerGRPC(t, runnerAddr)

	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)
	runnerClient := runnerv1.NewRunnerServiceClient(runnerConn)

	gatewayToken := gatewayAPIToken(t)
	gatewayIdentity := fetchGatewayIdentity(t, gatewayToken)
	identityID := gatewayIdentity.IdentityID
	threadsCtx := withIdentity(ctx, identityID)
	orgID := gatewayOrganizationID(t)
	modelID := gatewayModelID(t)

	agent := createAgent(t, threadsCtx, agentsClient, fmt.Sprintf("e2e-test-agent-start-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	var threadID string
	t.Cleanup(func() {
		if threadID != "" {
			archiveThread(t, threadsCtx, threadsClient, threadID)
		}
		deleteAgent(t, threadsCtx, agentsClient, agentID)
	})
	agentEnv := createAgentEnv(t, threadsCtx, agentsClient, agentID, "LLM_API_TOKEN", gatewayToken)
	agentEnvID := agentEnv.GetMeta().GetId()
	if agentEnvID == "" {
		t.Fatal("create agent env: missing id")
	}
	t.Cleanup(func() { deleteAgentEnv(t, threadsCtx, agentsClient, agentEnvID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID = thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}

	_ = sendMessage(t, threadsCtx, threadsClient, threadID, identityID, "e2e test message")

	labels := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentID,
		labelThreadID:  threadID,
	}

	pollCtx, pollCancel := context.WithTimeout(ctx, 90*time.Second)
	defer pollCancel()
	workloadID := ""
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			return err
		}
		if len(ids) != 1 {
			return fmt.Errorf("expected 1 workload, got %d", len(ids))
		}
		workloadID = ids[0]
		return nil
	}); err != nil {
		t.Fatalf("wait for workload: %v", err)
	}

	t.Cleanup(func() { cleanupWorkload(t, ctx, runnerClient, workloadID) })

	labelsResp, err := getWorkloadLabels(ctx, runnerClient, workloadID)
	if err != nil {
		t.Fatalf("get workload labels: %v", err)
	}
	assertLabel(t, labelsResp, labelManagedBy, managedByValue)
	assertLabel(t, labelsResp, labelAgentID, agentID)
	assertLabel(t, labelsResp, labelThreadID, threadID)
}
