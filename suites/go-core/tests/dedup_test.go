//go:build e2e && svc_agents_orchestrator

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

func TestNoDuplicateWorkloads(t *testing.T) {
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

	agent := createAgent(t, threadsCtx, agentsClient, fmt.Sprintf("e2e-test-agent-nodup-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentID) })
	createAgentEnv(t, threadsCtx, agentsClient, agentID, "LLM_API_TOKEN", gatewayToken)

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID := thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	for i := 1; i <= 3; i++ {
		sendMessage(t, threadsCtx, threadsClient, threadID, identityID, fmt.Sprintf("msg %d", i))
	}

	labels := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentID,
		labelThreadID:  threadID,
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		agentCleanupCtx := withAgentIdentity(cleanupCtx, agentID)
		ackAllUnackedMessagesBestEffort(t, agentCleanupCtx, threadsClient, agentID)
		ids, err := findWorkloadsByLabels(cleanupCtx, runnerClient, labels)
		if err != nil {
			t.Logf("cleanup: find workloads: %v", err)
			return
		}
		for _, workloadID := range ids {
			cleanupWorkload(t, cleanupCtx, runnerClient, workloadID)
		}
	})

	pollCtx, pollCancel := context.WithTimeout(ctx, 90*time.Second)
	defer pollCancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			return err
		}
		if len(ids) < 1 {
			return fmt.Errorf("expected at least 1 workload, got %d", len(ids))
		}
		return nil
	}); err != nil {
		t.Fatalf("wait for workload: %v", err)
	}

	dedupTimer := time.NewTimer(15 * time.Second)
	defer dedupTimer.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			t.Fatalf("find workloads: %v", err)
		}
		if len(ids) > 1 {
			t.Fatalf("expected at most 1 workload, got %d", len(ids))
		}
		select {
		case <-dedupTimer.C:
			return
		case <-ticker.C:
		}
	}
}
