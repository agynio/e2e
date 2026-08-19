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

	agent := createAgent(t, identityCtx, agentsClient, fmt.Sprintf("e2e-test-agent-start-%s", uuid.NewString()), modelID, orgID, codexRuntime)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() {
		cleanupAgentEnvs(t, identityCtx, agentsClient, orgID, agentID)
		deleteAgent(t, identityCtx, agentsClient, agentID)
	})
	createAgentEnv(t, identityCtx, agentsClient, agentID, "LLM_API_TOKEN", token)

	thread := createThread(t, identityCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID := thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, identityCtx, threadsClient, threadID) })

	// The thread creates the agent's instance, and the instance is what has an
	// inbox. Sending before it exists leaves the message with nowhere to land:
	// the orchestrator then sees an ACTIVE instance with nothing unacked and
	// wants no workload, which is exactly what it reported -- desired=0 against
	// an instance this test could see.
	instanceCtx, instanceCancel := context.WithTimeout(ctx, 60*time.Second)
	defer instanceCancel()
	if err := pollUntil(instanceCtx, pollInterval, func(ctx context.Context) error {
		listed, err := agentsClient.ListInstances(ctx, &agentsv1.ListInstancesRequest{
			OrganizationId: orgID,
			AgentId:        &agentID,
			PageSize:       10,
			StateIn:        []agentsv1.AgentInstanceState{agentsv1.AgentInstanceState_AGENT_INSTANCE_STATE_ACTIVE},
		})
		if err != nil {
			return err
		}
		if len(listed.GetInstances()) == 0 {
			return fmt.Errorf("no active instance for agent %s yet", agentID)
		}
		return nil
	}); err != nil {
		t.Fatalf("wait for the agent's instance: %v", err)
	}

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
		// Say what the platform thinks it should be running. The orchestrator
		// starts a workload for an ACTIVE instance with something unacked in its
		// inbox, so an empty desired set is either no instance or no message --
		// and which one it is decides where to look next.
		// And whether any of them has something unacked, which is the other half
		// of what the orchestrator selects on. Waiting for the instance did not
		// change the outcome, so the message is either not reaching the inbox or
		// not staying unacked -- and this says which.
		unacked := true
		unackedCount := -1
		if listed, listErr := agentsClient.ListInstances(ctx, &agentsv1.ListInstancesRequest{
			PageSize:   50,
			StateIn:    []agentsv1.AgentInstanceState{agentsv1.AgentInstanceState_AGENT_INSTANCE_STATE_ACTIVE},
			HasUnacked: &unacked,
		}); listErr == nil {
			unackedCount = len(listed.GetInstances())
		}

		instances := "unavailable"
		if listed, listErr := agentsClient.ListInstances(ctx, &agentsv1.ListInstancesRequest{
			PageSize: 50,
		}); listErr == nil {
			seen := []string{}
			for _, instance := range listed.GetInstances() {
				seen = append(seen, fmt.Sprintf("%s state=%s agent=%s",
					instance.GetMeta().GetId(), instance.GetState(), instance.GetAgentId()))
			}
			instances = fmt.Sprintf("%v", seen)
		} else {
			instances = listErr.Error()
		}
		t.Fatalf("wait for workload: %v; agent=%s thread=%s active-with-unacked=%d instances=%s",
			err, agentID, threadID, unackedCount, instances)
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
