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

func TestMultipleAgentsSeparateThreads(t *testing.T) {
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
	threadsCtx := setup.Context
	orgID := setup.OrganizationID
	token := setup.Token
	modelID := setup.ModelID

	agentA := createAgent(t, threadsCtx, agentsClient, fmt.Sprintf("e2e-test-agent-multi-a-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentB := createAgent(t, threadsCtx, agentsClient, fmt.Sprintf("e2e-test-agent-multi-b-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentAID := agentA.GetMeta().GetId()
	agentBID := agentB.GetMeta().GetId()
	if agentAID == "" || agentBID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() {
		cleanupAgentEnvs(t, threadsCtx, agentsClient, agentAID)
		deleteAgent(t, threadsCtx, agentsClient, agentAID)
	})
	t.Cleanup(func() {
		cleanupAgentEnvs(t, threadsCtx, agentsClient, agentBID)
		deleteAgent(t, threadsCtx, agentsClient, agentBID)
	})
	createAgentEnv(t, threadsCtx, agentsClient, agentAID, "LLM_API_TOKEN", token)
	createAgentEnv(t, threadsCtx, agentsClient, agentBID, "LLM_API_TOKEN", token)

	threadA := createThread(t, threadsCtx, threadsClient, orgID, []string{agentAID})
	threadB := createThread(t, threadsCtx, threadsClient, orgID, []string{agentBID})
	threadAID := threadA.GetId()
	threadBID := threadB.GetId()
	if threadAID == "" || threadBID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadAID) })
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadBID) })

	sendMessage(t, threadsCtx, threadsClient, threadAID, identityID, "multi agent message a")
	sendMessage(t, threadsCtx, threadsClient, threadBID, identityID, "multi agent message b")

	labelsA := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentAID,
		labelThreadID:  threadAID,
	}
	labelsB := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentBID,
		labelThreadID:  threadBID,
	}

	workloadAID := ""
	workloadBID := ""
	t.Cleanup(func() {
		if workloadAID != "" {
			cleanupWorkload(t, ctx, runnerClient, workloadAID)
		}
	})
	t.Cleanup(func() {
		if workloadBID != "" {
			cleanupWorkload(t, ctx, runnerClient, workloadBID)
		}
	})

	pollCtx, pollCancel := context.WithTimeout(ctx, 90*time.Second)
	defer pollCancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labelsA)
		if err != nil {
			return err
		}
		if len(ids) != 1 {
			return fmt.Errorf("expected 1 workload for agent A, got %d", len(ids))
		}
		workloadAID = ids[0]
		return nil
	}); err != nil {
		t.Fatalf("wait for workload A: %v", err)
	}

	pollCtxB, pollCancelB := context.WithTimeout(ctx, 90*time.Second)
	defer pollCancelB()
	if err := pollUntil(pollCtxB, pollInterval, func(ctx context.Context) error {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labelsB)
		if err != nil {
			return err
		}
		if len(ids) != 1 {
			return fmt.Errorf("expected 1 workload for agent B, got %d", len(ids))
		}
		workloadBID = ids[0]
		return nil
	}); err != nil {
		t.Fatalf("wait for workload B: %v", err)
	}

	if workloadAID == workloadBID {
		t.Fatalf("expected distinct workloads, got %s", workloadAID)
	}

	labelsRespA, err := getWorkloadLabels(ctx, runnerClient, workloadAID)
	if err != nil {
		t.Fatalf("get labels for workload A: %v", err)
	}
	labelsRespB, err := getWorkloadLabels(ctx, runnerClient, workloadBID)
	if err != nil {
		t.Fatalf("get labels for workload B: %v", err)
	}
	assertLabel(t, labelsRespA, labelManagedBy, managedByValue)
	assertLabel(t, labelsRespA, labelAgentID, agentAID)
	assertLabel(t, labelsRespA, labelThreadID, threadAID)
	assertLabel(t, labelsRespB, labelManagedBy, managedByValue)
	assertLabel(t, labelsRespB, labelAgentID, agentBID)
	assertLabel(t, labelsRespB, labelThreadID, threadBID)
}

func TestSameAgentMultipleThreads(t *testing.T) {
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
	threadsCtx := setup.Context
	orgID := setup.OrganizationID
	token := setup.Token
	modelID := setup.ModelID

	agent := createAgent(t, threadsCtx, agentsClient, fmt.Sprintf("e2e-test-agent-multi-thread-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() {
		cleanupAgentEnvs(t, threadsCtx, agentsClient, agentID)
		deleteAgent(t, threadsCtx, agentsClient, agentID)
	})
	createAgentEnv(t, threadsCtx, agentsClient, agentID, "LLM_API_TOKEN", token)

	threadA := createThread(t, threadsCtx, threadsClient, orgID, []string{agentID})
	threadB := createThread(t, threadsCtx, threadsClient, orgID, []string{agentID})
	threadAID := threadA.GetId()
	threadBID := threadB.GetId()
	if threadAID == "" || threadBID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadAID) })
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadBID) })

	sendMessage(t, threadsCtx, threadsClient, threadAID, identityID, "multi thread message a")
	sendMessage(t, threadsCtx, threadsClient, threadBID, identityID, "multi thread message b")

	labels := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentID,
	}

	workloadIDs := []string{}
	t.Cleanup(func() {
		for _, workloadID := range workloadIDs {
			cleanupWorkload(t, ctx, runnerClient, workloadID)
		}
	})

	pollCtx, pollCancel := context.WithTimeout(ctx, 90*time.Second)
	defer pollCancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			return err
		}
		if len(ids) != 2 {
			return fmt.Errorf("expected 2 workloads, got %d", len(ids))
		}
		workloadIDs = ids
		return nil
	}); err != nil {
		t.Fatalf("wait for workloads: %v", err)
	}

	threadIDs := map[string]bool{
		threadAID: true,
		threadBID: true,
	}
	foundThreads := map[string]bool{}
	for _, workloadID := range workloadIDs {
		labelsResp, err := getWorkloadLabels(ctx, runnerClient, workloadID)
		if err != nil {
			t.Fatalf("get labels for workload %s: %v", workloadID, err)
		}
		assertLabel(t, labelsResp, labelManagedBy, managedByValue)
		assertLabel(t, labelsResp, labelAgentID, agentID)
		threadID := labelsResp[labelThreadID]
		if !threadIDs[threadID] {
			t.Fatalf("unexpected thread id label %q", threadID)
		}
		foundThreads[threadID] = true
	}
	if len(foundThreads) != 2 {
		t.Fatalf("expected workloads for two threads, got %d", len(foundThreads))
	}
}
