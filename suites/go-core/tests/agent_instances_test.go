//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	"github.com/google/uuid"
)

// Every one of these covers a step that was broken in a released build and that
// no e2e exercised: the identity Threads forwards when it creates an instance,
// the OpenFGA type the write needs, and the default thread the row is written
// with. Each failed in a different service, and each looked like "the agent
// just never answers" from outside.

// An agent added to a thread gets an instance, and that instance carries the
// thread as its default -- the class policy is ORIGIN, and the thread that
// added it is the origin.
func TestInstanceInheritsOriginThread(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	gatewayToken := gatewayAPIToken(t)
	identityID := fetchGatewayIdentity(t, gatewayToken).IdentityID
	threadsCtx := withIdentity(ctx, identityID)
	orgID := gatewayOrganizationID(t)
	modelID := gatewayModelID(t)

	agent := createAgentWithOptions(t, threadsCtx, agentsClient, agentCreateOptions{
		Name:           fmt.Sprintf("e2e-instance-origin-%s", uuid.NewString()),
		Nickname:       nicknameFor("e2e-instance-origin"),
		Model:          modelID,
		OrganizationID: orgID,
		InitImage:      codexInitImage,
		DefaultThread:  agentsv1.AgentDefaultThread_AGENT_DEFAULT_THREAD_ORIGIN,
	})
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID := thread.GetId()
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	instance := waitForInstance(t, threadsCtx, agentsClient, agentID, orgID)
	t.Cleanup(func() { deleteInstance(t, threadsCtx, agentsClient, instance.GetMeta().GetId()) })
	if got := instance.GetDefaultThreadId(); got != threadID {
		t.Fatalf("expected the originating thread %s as the default, got %q", threadID, got)
	}
}

// The same act with a class that infers nothing leaves the instance without a
// default. Paired with the test above, this is what distinguishes "the policy
// was applied" from "the field is never written" -- the bug that shipped.
func TestInstanceWithoutOriginPolicyHasNoDefaultThread(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	gatewayToken := gatewayAPIToken(t)
	identityID := fetchGatewayIdentity(t, gatewayToken).IdentityID
	threadsCtx := withIdentity(ctx, identityID)
	orgID := gatewayOrganizationID(t)
	modelID := gatewayModelID(t)

	agent := createAgentWithOptions(t, threadsCtx, agentsClient, agentCreateOptions{
		Name:           fmt.Sprintf("e2e-instance-none-%s", uuid.NewString()),
		Nickname:       nicknameFor("e2e-instance-none"),
		Model:          modelID,
		OrganizationID: orgID,
		InitImage:      codexInitImage,
		DefaultThread:  agentsv1.AgentDefaultThread_AGENT_DEFAULT_THREAD_NONE,
	})
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, thread.GetId()) })

	instance := waitForInstance(t, threadsCtx, agentsClient, agentID, orgID)
	t.Cleanup(func() { deleteInstance(t, threadsCtx, agentsClient, instance.GetMeta().GetId()) })
	if got := instance.GetDefaultThreadId(); got != "" {
		t.Fatalf("expected no default thread, got %q", got)
	}
}

// A message to a thread becomes an inbox item addressed to the instance, not to
// the agent class. This is the delivery the whole feature exists for.
func TestThreadMessageReachesTheInstanceInbox(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	gatewayToken := gatewayAPIToken(t)
	identityID := fetchGatewayIdentity(t, gatewayToken).IdentityID
	threadsCtx := withIdentity(ctx, identityID)
	orgID := gatewayOrganizationID(t)
	modelID := gatewayModelID(t)

	agent := createAgent(t, threadsCtx, agentsClient,
		fmt.Sprintf("e2e-inbox-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID := thread.GetId()
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	instance := waitForInstance(t, threadsCtx, agentsClient, agentID, orgID)
	instanceID := instance.GetMeta().GetId()
	t.Cleanup(func() { deleteInstance(t, threadsCtx, agentsClient, instanceID) })

	body := fmt.Sprintf("inbox delivery %s", uuid.NewString())
	sendMessage(t, threadsCtx, threadsClient, threadID, identityID, body)

	pollCtx, pollCancel := context.WithTimeout(ctx, 60*time.Second)
	defer pollCancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		resp, err := agentsClient.GetUnackedInboxItems(withAgentIdentity(ctx, instanceID),
			&agentsv1.GetUnackedInboxItemsRequest{AgentInstanceId: instanceID, PageSize: 50})
		if err != nil {
			return err
		}
		for _, item := range resp.GetItems() {
			if strings.Contains(item.GetBody(), body) {
				if item.GetThreadId() != threadID {
					return fmt.Errorf("item names thread %q, want %q", item.GetThreadId(), threadID)
				}
				return nil
			}
		}
		return fmt.Errorf("message not in the inbox yet (%d items)", len(resp.GetItems()))
	}); err != nil {
		t.Fatalf("wait for inbox delivery: %v", err)
	}
}

// waitForInstance returns the instance created for an agent. Creation is lazy --
// adding the agent to a thread triggers it -- so this polls rather than assuming
// the row is there the moment the thread call returns.
func waitForInstance(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, agentID, organizationID string) *agentsv1.AgentInstance {
	t.Helper()
	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var found *agentsv1.AgentInstance
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		resp, err := client.ListInstances(ctx, &agentsv1.ListInstancesRequest{
			OrganizationId: organizationID,
			AgentId:        &agentID,
			PageSize:       50,
		})
		if err != nil {
			return err
		}
		if len(resp.GetInstances()) == 0 {
			return fmt.Errorf("no instance for agent %s yet", agentID)
		}
		found = resp.GetInstances()[0]
		return nil
	}); err != nil {
		t.Fatalf("wait for instance: %v", err)
	}
	return found
}

// deleteInstance removes an instance so its agent can be deleted -- an agent
// with a live instance refuses deletion, which otherwise leaves both behind on
// every run.
func deleteInstance(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, instanceID string) {
	t.Helper()
	if instanceID == "" {
		return
	}
	if _, err := client.DeleteInstance(ctx, &agentsv1.DeleteInstanceRequest{Id: instanceID}); err != nil {
		t.Logf("cleanup: delete instance %s: %v", instanceID, err)
	}
}

// An instance whose class sets a short idle limit is paused by the sweep, with
// a reason that tells "nobody used this" apart from "something went wrong".
// The sweep only reads instances whose class opted in, so a class without a
// limit is what proves it is the policy doing the work rather than a timer.
func TestIdleInstanceIsPausedByTheSweep(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	gatewayToken := gatewayAPIToken(t)
	identityID := fetchGatewayIdentity(t, gatewayToken).IdentityID
	threadsCtx := withIdentity(ctx, identityID)
	orgID := gatewayOrganizationID(t)
	modelID := gatewayModelID(t)

	// One second, so the instance is past its limit as soon as it exists.
	agent := createAgentWithOptions(t, threadsCtx, agentsClient, agentCreateOptions{
		Name:            fmt.Sprintf("e2e-idle-ttl-%s", uuid.NewString()),
		Nickname:        nicknameFor("e2e-idle-ttl"),
		Model:           modelID,
		OrganizationID:  orgID,
		InitImage:       codexInitImage,
		InstanceIdleTTL: "1s",
	})
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, thread.GetId()) })

	instance := waitForInstance(t, threadsCtx, agentsClient, agentID, orgID)
	instanceID := instance.GetMeta().GetId()
	t.Cleanup(func() { deleteInstance(t, threadsCtx, agentsClient, instanceID) })

	// The sweep runs on its own schedule, so this waits for a tick rather than
	// assuming one has happened.
	pollCtx, pollCancel := context.WithTimeout(ctx, 180*time.Second)
	defer pollCancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		resp, err := agentsClient.GetInstance(ctx, &agentsv1.GetInstanceRequest{Id: instanceID})
		if err != nil {
			return err
		}
		got := resp.GetInstance()
		if got.GetState() != agentsv1.AgentInstanceState_AGENT_INSTANCE_STATE_PAUSED {
			return fmt.Errorf("state is %s, want paused", got.GetState())
		}
		if reason := got.GetPauseReason(); reason != "idle_ttl_exceeded" {
			return fmt.Errorf("pause reason %q, want idle_ttl_exceeded", reason)
		}
		return nil
	}); err != nil {
		t.Fatalf("wait for the idle sweep: %v", err)
	}

	// Resuming clears the reason, which is what makes the pause recoverable
	// rather than a quiet death.
	if _, err := agentsClient.ResumeInstance(threadsCtx, &agentsv1.ResumeInstanceRequest{Id: instanceID}); err != nil {
		t.Fatalf("resume instance: %v", err)
	}
	resumed, err := agentsClient.GetInstance(threadsCtx, &agentsv1.GetInstanceRequest{Id: instanceID})
	if err != nil {
		t.Fatalf("get instance after resume: %v", err)
	}
	if got := resumed.GetInstance(); got.GetState() != agentsv1.AgentInstanceState_AGENT_INSTANCE_STATE_ACTIVE {
		t.Fatalf("state after resume is %s, want active", got.GetState())
	}
	if reason := resumed.GetInstance().GetPauseReason(); reason != "" {
		t.Fatalf("pause reason after resume is %q, want empty", reason)
	}
}

// An agent whose class sets no idle limit is left alone. Without this the test
// above would pass just as well against a sweep that paused everything.
func TestInstanceWithoutIdleTTLIsNotPaused(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	gatewayToken := gatewayAPIToken(t)
	identityID := fetchGatewayIdentity(t, gatewayToken).IdentityID
	threadsCtx := withIdentity(ctx, identityID)
	orgID := gatewayOrganizationID(t)
	modelID := gatewayModelID(t)

	agent := createAgent(t, threadsCtx, agentsClient,
		fmt.Sprintf("e2e-no-idle-ttl-%s", uuid.NewString()), modelID, orgID, codexInitImage)
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, thread.GetId()) })

	instance := waitForInstance(t, threadsCtx, agentsClient, agentID, orgID)
	instanceID := instance.GetMeta().GetId()
	t.Cleanup(func() { deleteInstance(t, threadsCtx, agentsClient, instanceID) })

	// Long enough for several sweep ticks to have come and gone.
	time.Sleep(90 * time.Second)

	resp, err := agentsClient.GetInstance(threadsCtx, &agentsv1.GetInstanceRequest{Id: instanceID})
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if got := resp.GetInstance().GetState(); got != agentsv1.AgentInstanceState_AGENT_INSTANCE_STATE_ACTIVE {
		t.Fatalf("state is %s, want active -- the sweep took an instance whose class set no limit", got)
	}
}

// An explicit default_thread_id wins over the class policy. This is what
// `agyn agents instantiate --default-thread` sets, and the reason the field
// exists separately from the creation context: naming a thread is a decision,
// while the context only reports which thread happened to be involved.
func TestExplicitDefaultThreadOverridesThePolicy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	gatewayToken := gatewayAPIToken(t)
	identityID := fetchGatewayIdentity(t, gatewayToken).IdentityID
	threadsCtx := withIdentity(ctx, identityID)
	orgID := gatewayOrganizationID(t)
	modelID := gatewayModelID(t)

	// NONE, so the policy would leave the default unset and only the explicit
	// value can put one there.
	agent := createAgentWithOptions(t, threadsCtx, agentsClient, agentCreateOptions{
		Name:           fmt.Sprintf("e2e-explicit-thread-%s", uuid.NewString()),
		Nickname:       nicknameFor("e2e-explicit-thread"),
		Model:          modelID,
		OrganizationID: orgID,
		InitImage:      codexInitImage,
		DefaultThread:  agentsv1.AgentDefaultThread_AGENT_DEFAULT_THREAD_NONE,
	})
	agentID := agent.GetMeta().GetId()
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentID) })

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID})
	threadID := thread.GetId()
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	resp, err := agentsClient.CreateInstance(threadsCtx, &agentsv1.CreateInstanceRequest{
		AgentId:         agentID,
		DefaultThreadId: &threadID,
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	instance := resp.GetInstance()
	t.Cleanup(func() { deleteInstance(t, threadsCtx, agentsClient, instance.GetMeta().GetId()) })

	if got := instance.GetDefaultThreadId(); got != threadID {
		t.Fatalf("expected the named thread %s, got %q", threadID, got)
	}
}
