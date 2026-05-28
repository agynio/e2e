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

	setup := newWorkflowGatewaySetup(t, ctx)
	identityID := setup.IdentityID
	identityCtx := setup.Context
	orgID := setup.OrganizationID
	token := setup.Token
	modelID := setup.ModelID

	agent := createAgent(t, identityCtx, agentsClient, fmt.Sprintf("e2e-test-agent-start-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() {
		cleanupAgentEnvs(t, identityCtx, agentsClient, agentID)
		deleteAgent(t, identityCtx, agentsClient, agentID)
	})
	createAgentEnv(t, identityCtx, agentsClient, agentID, "LLM_API_TOKEN", token)

	thread := createThread(t, identityCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID := thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, identityCtx, threadsClient, threadID) })

	_ = sendMessage(t, identityCtx, threadsClient, threadID, identityID, "e2e test message")

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
