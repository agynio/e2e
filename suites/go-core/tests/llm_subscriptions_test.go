//go:build e2e && (svc_llm || svc_llm_proxy)

package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	authorizationv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/authorization/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	secretsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/secrets/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Covers the control plane of native LLM mode: a subscription is a reference to
// a secret, and everything downstream of it -- the binding the proxy resolves,
// the allowlist it enforces, and the guard that stops the secret disappearing
// from under it -- is derived rather than copied.

const (
	subscriptionTestToken = "e2e-subscription-token"

	// An environment needs an image to be created at all. Nothing here runs a
	// workload, so the reference only has to be well-formed.
	environmentPlaceholderImage = "ghcr.io/agynio/devcontainer:latest"
)

type subscriptionFixture struct {
	ownerCtx       context.Context
	identityID     string
	organizationID string
	environmentID  string
	agentID        string
	llm            llmv1.LLMServiceClient
	agents         agentsv1.AgentsServiceClient
	secrets        secretsv1.SecretsServiceClient
	authorization  authorizationv1.AuthorizationServiceClient
}

func setupSubscriptionFixture(t *testing.T, ctx context.Context) subscriptionFixture {
	t.Helper()

	usersClient := usersv1.NewUsersServiceClient(dialGRPC(t, usersAddr))
	orgsClient := organizationsv1.NewOrganizationsServiceClient(dialGRPC(t, orgsAddr))
	llmClient := llmv1.NewLLMServiceClient(dialGRPC(t, llmAddr))
	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	secretsClient := secretsv1.NewSecretsServiceClient(dialGRPC(t, secretsAddr))
	authzClient := newLLMProxyAuthorizationClient(t)

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	ownerCtx := withIdentity(ctx, identityID)
	organizationID := createTestOrganization(t, ctx, orgsClient, identityID)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := orgsClient.DeleteOrganization(withIdentity(cleanupCtx, identityID), &organizationsv1.DeleteOrganizationRequest{
			Id: organizationID,
		}); err != nil {
			t.Logf("cleanup: delete organization %s: %v", organizationID, err)
		}
	})

	environmentID := createSubscriptionEnvironment(t, ownerCtx, agentsClient, identityID, organizationID)

	// An agent-scoped attachment is authorized against the agent, not the
	// environment, so the fixture needs an agent object the caller may edit.
	// It never runs: nothing here starts a workload.
	agentID := uuid.NewString()
	grantSubscriptionTargetAccess(t, ctx, authzClient, identityID, organizationID, "agent:"+agentID)

	return subscriptionFixture{
		ownerCtx:       ownerCtx,
		identityID:     identityID,
		organizationID: organizationID,
		environmentID:  environmentID,
		agentID:        agentID,
		llm:            llmClient,
		agents:         agentsClient,
		secrets:        secretsClient,
		authorization:  authzClient,
	}
}

func createSubscriptionEnvironment(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, identityID, organizationID string) string {
	t.Helper()
	created, err := client.CreateEnvironment(ctx, &agentsv1.CreateEnvironmentRequest{
		OrganizationId: organizationID,
		Name:           "e2e-native-" + uuid.NewString(),
		RunnerId:       uuid.NewString(),
		// Deprecated inline image rather than a catalog entry: these tests never
		// start a workload, and registering an image would test the catalog.
		Image:        environmentPlaceholderImage,
		Availability: agentsv1.EnvironmentAvailability_ENVIRONMENT_AVAILABILITY_INTERNAL,
		LlmMode:      agentsv1.LLMMode_LLM_MODE_NATIVE,
		// The allowlist is the environment's, and the proxy reads it through
		// ResolveSubscription rather than calling Agents on the request path.
		LlmAllowedModels: []string{"claude-sonnet-4-6"},
	})
	if err != nil {
		t.Fatalf("create native environment: %v", err)
	}
	environmentID := created.GetEnvironment().GetMeta().GetId()
	if environmentID == "" {
		t.Fatal("create native environment: missing id")
	}
	if created.GetEnvironment().GetLlmMode() != agentsv1.LLMMode_LLM_MODE_NATIVE {
		t.Fatalf("environment came back in mode %v", created.GetEnvironment().GetLlmMode())
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := client.DeleteEnvironment(withIdentity(cleanupCtx, identityID), &agentsv1.DeleteEnvironmentRequest{
			Id: environmentID,
		}); err != nil {
			t.Logf("cleanup: delete environment %s: %v", environmentID, err)
		}
	})
	return environmentID
}

func grantSubscriptionTargetAccess(t *testing.T, ctx context.Context, client authorizationv1.AuthorizationServiceClient, identityID, organizationID, object string) {
	t.Helper()
	// can_edit_config is derived (owner or maintainer or owner from org), so it
	// cannot be written. Writing the roles it derives from is what grants it:
	// the caller owns the organization, and org ownership reaches the agent
	// through the org tuple.
	tuples := []*authorizationv1.TupleKey{
		{User: "identity:" + identityID, Relation: "owner", Object: object},
		{User: "organization:" + organizationID, Relation: "org", Object: object},
	}
	if _, err := client.Write(llmProxyAdminContext(ctx), &authorizationv1.WriteRequest{Writes: tuples}); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("grant access on %s: %v", object, err)
		}
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := client.Write(llmProxyAdminContext(cleanupCtx), &authorizationv1.WriteRequest{Deletes: tuples}); err != nil {
			t.Logf("cleanup: revoke access on %s: %v", object, err)
		}
	})
}

func createSubscriptionSecret(t *testing.T, ctx context.Context, client secretsv1.SecretsServiceClient, identityID, organizationID string) string {
	t.Helper()
	resp, err := client.CreateSecret(ctx, &secretsv1.CreateSecretRequest{
		Title:          "e2e-subscription-" + uuid.NewString(),
		Description:    "E2E native mode subscription token",
		OrganizationId: organizationID,
		Value:          subscriptionTestToken,
	})
	if err != nil {
		t.Fatalf("create subscription secret: %v", err)
	}
	secretID := resp.GetSecret().GetMeta().GetId()
	if secretID == "" {
		t.Fatal("create subscription secret: missing id")
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := client.DeleteSecret(withIdentity(cleanupCtx, identityID), &secretsv1.DeleteSecretRequest{
			Id: secretID,
		}); err != nil && status.Code(err) != codes.NotFound {
			t.Logf("cleanup: delete secret %s: %v", secretID, err)
		}
	})
	return secretID
}

func createSubscription(t *testing.T, ctx context.Context, client llmv1.LLMServiceClient, identityID, organizationID, secretID string, vendor llmv1.Vendor) *llmv1.Subscription {
	t.Helper()
	resp, err := client.CreateSubscription(ctx, &llmv1.CreateSubscriptionRequest{
		OrganizationId: organizationID,
		Name:           "e2e-subscription-" + uuid.NewString(),
		Vendor:         vendor,
		SecretId:       secretID,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	subscription := resp.GetSubscription()
	if subscription.GetMeta().GetId() == "" {
		t.Fatal("create subscription: missing id")
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := client.DeleteSubscription(withIdentity(cleanupCtx, identityID), &llmv1.DeleteSubscriptionRequest{
			Id: subscription.GetMeta().GetId(),
		}); err != nil && status.Code(err) != codes.NotFound {
			t.Logf("cleanup: delete subscription %s: %v", subscription.GetMeta().GetId(), err)
		}
	})
	return subscription
}

func attachSubscriptionToEnvironment(t *testing.T, ctx context.Context, client llmv1.LLMServiceClient, subscriptionID, environmentID string) *llmv1.SubscriptionAttachment {
	t.Helper()
	resp, err := client.CreateSubscriptionAttachment(ctx, &llmv1.CreateSubscriptionAttachmentRequest{
		SubscriptionId: subscriptionID,
		Target:         &llmv1.CreateSubscriptionAttachmentRequest_EnvironmentId{EnvironmentId: environmentID},
	})
	if err != nil {
		t.Fatalf("attach subscription to environment: %v", err)
	}
	return resp.GetSubscriptionAttachment()
}

// TestSubscriptionResolvesToATokenAndAnAllowlist is the whole native-mode
// binding in one pass: what the proxy asks for at connection time, and what it
// gets back.
func TestSubscriptionResolvesToATokenAndAnAllowlist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	secretID := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	subscription := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, secretID, llmv1.Vendor_VENDOR_CLAUDE)
	attachment := attachSubscriptionToEnvironment(t, fixture.ownerCtx, fixture.llm, subscription.GetMeta().GetId(), fixture.environmentID)

	// Denormalized onto the attachment so the orchestrator gets role attributes
	// without a second call, and carrying the variable the placeholder goes in.
	if attachment.GetVendor() != llmv1.Vendor_VENDOR_CLAUDE {
		t.Fatalf("attachment vendor = %v", attachment.GetVendor())
	}
	if attachment.GetPlaceholderEnv() != "CLAUDE_CODE_OAUTH_TOKEN" {
		t.Fatalf("attachment placeholder env = %q", attachment.GetPlaceholderEnv())
	}

	resolved, err := fixture.llm.ResolveSubscription(fixture.ownerCtx, &llmv1.ResolveSubscriptionRequest{
		EnvironmentId: fixture.environmentID,
		Vendor:        llmv1.Vendor_VENDOR_CLAUDE,
	})
	if err != nil {
		t.Fatalf("resolve subscription: %v", err)
	}
	if resolved.GetToken() != subscriptionTestToken {
		t.Fatalf("resolved token = %q, want the secret's value", resolved.GetToken())
	}
	if resolved.GetSubscriptionId() != subscription.GetMeta().GetId() {
		t.Fatalf("resolved subscription = %q, want %q", resolved.GetSubscriptionId(), subscription.GetMeta().GetId())
	}
	if resolved.GetUpstreamEndpoint() != "https://api.anthropic.com" {
		t.Fatalf("resolved upstream = %q", resolved.GetUpstreamEndpoint())
	}
	if len(resolved.GetAllowedModels()) != 1 || resolved.GetAllowedModels()[0] != "claude-sonnet-4-6" {
		t.Fatalf("resolved allowlist = %v, want the environment's", resolved.GetAllowedModels())
	}
	// The placeholder variable name is on the attachment, where the orchestrator
	// reads it. Nothing that reaches the proxy's request path carries it.
	if strings.Contains(resolved.String(), "CLAUDE_CODE_OAUTH_TOKEN") {
		t.Fatalf("resolve response leaks the placeholder variable: %s", resolved.String())
	}
}

// A sandbox runs no agent class, so the environment scope has to answer on its
// own -- and an agent-scoped attachment has to win when there is one.
func TestSubscriptionAgentScopeShadowsEnvironmentScope(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	environmentSecret := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	environmentSub := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, environmentSecret, llmv1.Vendor_VENDOR_CLAUDE)
	attachSubscriptionToEnvironment(t, fixture.ownerCtx, fixture.llm, environmentSub.GetMeta().GetId(), fixture.environmentID)

	agentSecret := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	agentSub := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, agentSecret, llmv1.Vendor_VENDOR_CLAUDE)
	if _, err := fixture.llm.CreateSubscriptionAttachment(fixture.ownerCtx, &llmv1.CreateSubscriptionAttachmentRequest{
		SubscriptionId: agentSub.GetMeta().GetId(),
		Target:         &llmv1.CreateSubscriptionAttachmentRequest_AgentId{AgentId: fixture.agentID},
	}); err != nil {
		t.Fatalf("attach subscription to agent: %v", err)
	}

	sandbox, err := fixture.llm.ResolveSubscription(fixture.ownerCtx, &llmv1.ResolveSubscriptionRequest{
		EnvironmentId: fixture.environmentID,
		Vendor:        llmv1.Vendor_VENDOR_CLAUDE,
	})
	if err != nil {
		t.Fatalf("resolve for a sandbox: %v", err)
	}
	if sandbox.GetSubscriptionId() != environmentSub.GetMeta().GetId() {
		t.Fatalf("sandbox resolved to %q, want the environment's subscription", sandbox.GetSubscriptionId())
	}

	agentScoped, err := fixture.llm.ResolveSubscription(fixture.ownerCtx, &llmv1.ResolveSubscriptionRequest{
		EnvironmentId: fixture.environmentID,
		AgentId:       fixture.agentID,
		Vendor:        llmv1.Vendor_VENDOR_CLAUDE,
	})
	if err != nil {
		t.Fatalf("resolve for an agent: %v", err)
	}
	if agentScoped.GetSubscriptionId() != agentSub.GetMeta().GetId() {
		t.Fatalf("agent resolved to %q, want the agent's subscription", agentScoped.GetSubscriptionId())
	}
}

// One vendor, one subscription per target: a second attachment for the same
// vendor would make the binding a coin flip.
func TestSubscriptionAttachmentIsUniquePerVendorAndTarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	firstSecret := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	first := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, firstSecret, llmv1.Vendor_VENDOR_CLAUDE)
	attachSubscriptionToEnvironment(t, fixture.ownerCtx, fixture.llm, first.GetMeta().GetId(), fixture.environmentID)

	secondSecret := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	second := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, secondSecret, llmv1.Vendor_VENDOR_CLAUDE)
	_, err := fixture.llm.CreateSubscriptionAttachment(fixture.ownerCtx, &llmv1.CreateSubscriptionAttachmentRequest{
		SubscriptionId: second.GetMeta().GetId(),
		Target:         &llmv1.CreateSubscriptionAttachmentRequest_EnvironmentId{EnvironmentId: fixture.environmentID},
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists on a second claude attachment, got %v", err)
	}
}

// The secret is the subscription's only copy of the token, so deleting it would
// break every workload bound to it -- silently, at the next connection.
func TestSecretReferencedByASubscriptionCannotBeDeleted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	secretID := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	subscription := createSubscription(t, fixture.ownerCtx, fixture.llm, fixture.identityID, fixture.organizationID, secretID, llmv1.Vendor_VENDOR_CLAUDE)

	_, err := fixture.secrets.DeleteSecret(fixture.ownerCtx, &secretsv1.DeleteSecretRequest{Id: secretID})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition deleting a referenced secret, got %v", err)
	}
	// Naming the blocker is the point: an operator has to know what to detach.
	if !strings.Contains(strings.ToLower(err.Error()), "subscription") {
		t.Fatalf("delete error does not name the subscription: %v", err)
	}

	if _, err := fixture.llm.DeleteSubscription(fixture.ownerCtx, &llmv1.DeleteSubscriptionRequest{
		Id: subscription.GetMeta().GetId(),
	}); err != nil {
		t.Fatalf("delete subscription: %v", err)
	}
	if _, err := fixture.secrets.DeleteSecret(fixture.ownerCtx, &secretsv1.DeleteSecretRequest{Id: secretID}); err != nil {
		t.Fatalf("delete secret once unreferenced: %v", err)
	}
}

// Codex has no stable public endpoint to bind to yet. Refusing the subscription
// at creation is the honest answer; accepting one that can never resolve is not.
func TestCodexSubscriptionIsRefusedUntilItsBindingShips(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	fixture := setupSubscriptionFixture(t, ctx)

	secretID := createSubscriptionSecret(t, fixture.ownerCtx, fixture.secrets, fixture.identityID, fixture.organizationID)
	_, err := fixture.llm.CreateSubscription(fixture.ownerCtx, &llmv1.CreateSubscriptionRequest{
		OrganizationId: fixture.organizationID,
		Name:           "e2e-codex-" + uuid.NewString(),
		Vendor:         llmv1.Vendor_VENDOR_CODEX,
		SecretId:       secretID,
	})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected Unimplemented for a codex subscription, got %v", err)
	}
}
