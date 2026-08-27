//go:build e2e && (svc_llm || svc_llm_proxy)

package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	zitimgmtv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Covers the rest of native mode's control plane: what an identity carries, what
// an agent may reference, and what a mode change costs once agents depend on it.
// The binding itself is in llm_subscriptions_test.go.

// The proxy resolves a subscription from the caller's identity alone. If the
// identity does not carry the environment, the proxy either calls Agents on the
// request path or trusts a value the workload asserts about itself.
//
// agent_id on the identity is the agent *class*, not the instance: a
// subscription attaches to the class, so that is what the lookup needs.
func TestWorkloadIdentityCarriesAgentAndEnvironment(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := zitimgmtv1.NewZitiManagementServiceClient(dialGRPC(t, zitiManagementAddr(t)))

	agentID := uuid.NewString()
	workloadID := uuid.NewString()
	agentClassID := uuid.NewString()
	environmentID := uuid.NewString()

	created, err := client.CreateAgentIdentity(ctx, &zitimgmtv1.CreateAgentIdentityRequest{
		AgentId:       agentID,
		WorkloadId:    workloadID,
		AgentClassId:  agentClassID,
		EnvironmentId: environmentID,
	})
	if err != nil {
		t.Fatalf("create agent identity: %v", err)
	}
	zitiIdentityID := created.GetZitiIdentityId()
	if zitiIdentityID == "" {
		t.Fatal("create agent identity: missing ziti identity id")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := client.DeleteIdentity(cleanupCtx, &zitimgmtv1.DeleteIdentityRequest{
			ZitiIdentityId: zitiIdentityID,
		}); err != nil {
			t.Logf("cleanup: delete identity %s: %v", zitiIdentityID, err)
		}
	})

	resolved, err := client.ResolveIdentity(ctx, &zitimgmtv1.ResolveIdentityRequest{
		ZitiIdentityId: zitiIdentityID,
	})
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if resolved.GetAgentId() != agentClassID {
		t.Fatalf("resolved agent_id = %q, want the class %q", resolved.GetAgentId(), agentClassID)
	}
	if resolved.GetEnvironmentId() != environmentID {
		t.Fatalf("resolved environment_id = %q, want %q", resolved.GetEnvironmentId(), environmentID)
	}
	if resolved.GetWorkloadId() != workloadID {
		t.Fatalf("resolved workload_id = %q, want %q", resolved.GetWorkloadId(), workloadID)
	}
}

// A sandbox runs an environment with no agent behind it, so agent_id has to be
// absent without the environment going with it -- that pair is exactly what the
// proxy uses to fall back to the environment-scoped subscription.
func TestSandboxIdentityCarriesEnvironmentWithoutAgent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := zitimgmtv1.NewZitiManagementServiceClient(dialGRPC(t, zitiManagementAddr(t)))
	environmentID := uuid.NewString()

	created, err := client.CreateSandboxIdentity(ctx, &zitimgmtv1.CreateSandboxIdentityRequest{
		SandboxId:      uuid.NewString(),
		OwnerId:        uuid.NewString(),
		OrganizationId: uuid.NewString(),
		WorkloadId:     uuid.NewString(),
		EnvironmentId:  environmentID,
	})
	if err != nil {
		t.Fatalf("create sandbox identity: %v", err)
	}
	zitiIdentityID := created.GetZitiIdentityId()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := client.DeleteIdentity(cleanupCtx, &zitimgmtv1.DeleteIdentityRequest{
			ZitiIdentityId: zitiIdentityID,
		}); err != nil {
			t.Logf("cleanup: delete identity %s: %v", zitiIdentityID, err)
		}
	})

	resolved, err := client.ResolveIdentity(ctx, &zitimgmtv1.ResolveIdentityRequest{
		ZitiIdentityId: zitiIdentityID,
	})
	if err != nil {
		t.Fatalf("resolve sandbox identity: %v", err)
	}
	if resolved.GetEnvironmentId() != environmentID {
		t.Fatalf("resolved environment_id = %q, want %q", resolved.GetEnvironmentId(), environmentID)
	}
	if resolved.GetAgentId() != "" {
		t.Fatalf("sandbox identity carries agent_id %q", resolved.GetAgentId())
	}
}

// model and model_name are required-and-exclusive against the environment's
// mode. Validating at configure time is the point: the alternative surfaces the
// mismatch at the agent's first model call.
func TestAgentModelReferenceMustMatchTheEnvironmentMode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	platformEnvironmentID := createPlatformEnvironment(t, fixture.ownerCtx, fixture.agents, fixture.identityID, fixture.organizationID)

	cases := []struct {
		name          string
		environmentID string
		model         string
		modelName     string
		wantErr       bool
		wantContains  string
	}{
		{
			name:          "native rejects a platform model UUID",
			environmentID: fixture.environmentID,
			model:         uuid.NewString(),
			wantErr:       true,
			wantContains:  "native",
		},
		{
			name:          "native accepts a vendor model name",
			environmentID: fixture.environmentID,
			modelName:     "claude-sonnet-4-6",
		},
		{
			name:          "platform rejects a vendor model name",
			environmentID: platformEnvironmentID,
			modelName:     "claude-sonnet-4-6",
			wantErr:       true,
			wantContains:  "platform",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := &agentsv1.CreateAgentRequest{
				OrganizationId: fixture.organizationID,
				Name:           "e2e-agent-" + uuid.NewString(),
				EnvironmentId:  tc.environmentID,
				Model:          tc.model,
				ModelName:      tc.modelName,
				Availability:   agentsv1.AgentAvailability_AGENT_AVAILABILITY_INTERNAL,
			}
			created, err := fixture.agents.CreateAgent(fixture.ownerCtx, request)
			if tc.wantErr {
				if status.Code(err) != codes.InvalidArgument {
					t.Fatalf("expected InvalidArgument, got %v", err)
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.wantContains) {
					t.Fatalf("error does not name the mode: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("create agent: %v", err)
			}
			agentID := created.GetAgent().GetMeta().GetId()
			t.Cleanup(func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cleanupCancel()
				if _, err := fixture.agents.DeleteAgent(withIdentity(cleanupCtx, fixture.identityID), &agentsv1.DeleteAgentRequest{
					Id: agentID,
				}); err != nil {
					t.Logf("cleanup: delete agent %s: %v", agentID, err)
				}
			})
			if created.GetAgent().GetModelName() != tc.modelName {
				t.Fatalf("model_name came back %q, want %q", created.GetAgent().GetModelName(), tc.modelName)
			}
		})
	}
}

// Switching an environment's mode invalidates every referencing agent's model
// reference at once, so it is refused while any agent references it -- and the
// refusal names them, because an operator has to know what to change.
func TestLLMModeCannotChangeWhileAgentsReferenceTheEnvironment(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	agentName := "e2e-agent-" + uuid.NewString()
	created, err := fixture.agents.CreateAgent(fixture.ownerCtx, &agentsv1.CreateAgentRequest{
		OrganizationId: fixture.organizationID,
		Name:           agentName,
		EnvironmentId:  fixture.environmentID,
		ModelName:      "claude-sonnet-4-6",
		Availability:   agentsv1.AgentAvailability_AGENT_AVAILABILITY_INTERNAL,
	})
	if err != nil {
		t.Fatalf("create agent in the native environment: %v", err)
	}
	agentID := created.GetAgent().GetMeta().GetId()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := fixture.agents.DeleteAgent(withIdentity(cleanupCtx, fixture.identityID), &agentsv1.DeleteAgentRequest{
			Id: agentID,
		}); err != nil {
			t.Logf("cleanup: delete agent %s: %v", agentID, err)
		}
	})

	platform := agentsv1.LLMMode_LLM_MODE_PLATFORM
	_, err = fixture.agents.UpdateEnvironment(fixture.ownerCtx, &agentsv1.UpdateEnvironmentRequest{
		Id:      fixture.environmentID,
		LlmMode: &platform,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition changing llm_mode, got %v", err)
	}
	if !strings.Contains(err.Error(), agentName) {
		t.Fatalf("refusal does not name the referencing agent: %v", err)
	}
}

// The allowlist lives on the environment and reaches the proxy only through
// ResolveSubscription -- that is what keeps Agents off the proxy's request path.
// Changing it has to be visible on the next resolve.
func TestAllowedModelsChangeIsVisibleToResolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	secretID := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	subscription := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, secretID, llmv1.Vendor_VENDOR_ANTHROPIC)
	attachSubscriptionToEnvironment(t, fixture.ownerCtx, fixture.llm, subscription.GetMeta().GetId(), fixture.environmentID)

	if _, err := fixture.agents.UpdateEnvironment(fixture.ownerCtx, &agentsv1.UpdateEnvironmentRequest{
		Id:               fixture.environmentID,
		LlmAllowedModels: []string{"claude-opus-4-6", "claude-haiku-4-5"},
	}); err != nil {
		t.Fatalf("tighten the allowlist: %v", err)
	}

	resolved, err := fixture.llm.ResolveSubscription(fixture.ownerCtx, &llmv1.ResolveSubscriptionRequest{
		EnvironmentId: fixture.environmentID,
		Vendor:        llmv1.Vendor_VENDOR_ANTHROPIC,
	})
	if err != nil {
		t.Fatalf("resolve after the allowlist change: %v", err)
	}
	got := resolved.GetAllowedModels()
	if len(got) != 2 || got[0] != "claude-opus-4-6" || got[1] != "claude-haiku-4-5" {
		t.Fatalf("resolved allowlist = %v, want the environment's new list", got)
	}
}

// Deleting a subscription that is still attached would break every workload
// bound to it at its next connection, silently.
func TestSubscriptionCannotBeDeletedWhileAttached(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	secretID := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	subscription := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, secretID, llmv1.Vendor_VENDOR_ANTHROPIC)
	attachment := attachSubscriptionToEnvironment(t, fixture.ownerCtx, fixture.llm, subscription.GetMeta().GetId(), fixture.environmentID)

	_, err := fixture.llm.DeleteSubscription(fixture.ownerCtx, &llmv1.DeleteSubscriptionRequest{
		Id: subscription.GetMeta().GetId(),
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition deleting an attached subscription, got %v", err)
	}
	// Named, not counted: an operator who has to go find them learns nothing.
	if !strings.Contains(err.Error(), fixture.environmentID) {
		t.Fatalf("refusal does not name the target: %v", err)
	}

	if _, err := fixture.llm.DeleteSubscriptionAttachment(fixture.ownerCtx, &llmv1.DeleteSubscriptionAttachmentRequest{
		Id: attachment.GetMeta().GetId(),
	}); err != nil {
		t.Fatalf("detach: %v", err)
	}
	if _, err := fixture.llm.DeleteSubscription(fixture.ownerCtx, &llmv1.DeleteSubscriptionRequest{
		Id: subscription.GetMeta().GetId(),
	}); err != nil {
		t.Fatalf("delete once detached: %v", err)
	}
}

// A native environment with nothing attached resolves to NOT_FOUND rather than
// to some other vendor's subscription, which is what makes the orchestrator's
// pre-flight refusal the only thing standing between a workload and a failed
// first call.
func TestResolveWithoutAnAttachmentIsNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	_, err := fixture.llm.ResolveSubscription(fixture.ownerCtx, &llmv1.ResolveSubscriptionRequest{
		EnvironmentId: fixture.environmentID,
		Vendor:        llmv1.Vendor_VENDOR_ANTHROPIC,
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound resolving an unattached environment, got %v", err)
	}
}

func createPlatformEnvironment(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, identityID, organizationID string) string {
	t.Helper()
	runtimeImageID, runtimeImageTag := platformAgentRuntime(t, ctx, organizationID, agnRuntime)
	created, err := client.CreateEnvironment(ctx, &agentsv1.CreateEnvironmentRequest{
		OrganizationId: organizationID,
		Name:           "e2e-platform-" + uuid.NewString(),
		RunnerId:       uuid.NewString(),
		// Deprecated inline image rather than a catalog entry: these tests never
		// start a workload, and registering an image would test the catalog.
		Image: environmentPlaceholderImage,
		// The runtime image is not optional even here. An environment names the
		// agent CLI an agent runs, so one without it can host no agent at all --
		// CreateAgent refuses it long before anything would start a workload.
		AgentRuntimeImageId:  runtimeImageID,
		AgentRuntimeImageTag: runtimeImageTag,
		Availability:         agentsv1.EnvironmentAvailability_ENVIRONMENT_AVAILABILITY_INTERNAL,
		// Left unset on purpose: an environment that predates native mode
		// carries no mode at all and must read as platform.
	})
	if err != nil {
		t.Fatalf("create platform environment: %v", err)
	}
	environmentID := created.GetEnvironment().GetMeta().GetId()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := client.DeleteEnvironment(withIdentity(cleanupCtx, identityID), &agentsv1.DeleteEnvironmentRequest{
			Id: environmentID,
		}); err != nil {
			t.Logf("cleanup: delete environment %s: %v", environmentID, err)
		}
	})
	return environmentID
}
