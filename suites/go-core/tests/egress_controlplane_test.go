//go:build e2e && svc_egress

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	authorizationv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/authorization/v1"
	egressv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/egress/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	secretsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/secrets/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultEgressAddr            = "egress:50051"
	egressGatewayTestTimeout     = 3 * time.Minute
	egressRulePropagationTimeout = 45 * time.Second
	egressExpectedSecretValue    = "e2e-egress-secret-value"
)

func TestEgressGatewayFeaturePath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), egressGatewayTestTimeout)
	t.Cleanup(cancel)

	fixture := setupEgressFixture(t, ctx)
	secret := createEgressSecret(t, fixture.userCtx, fixture.secrets, fixture.organizationID)
	secretID := secret.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressSecret(t, fixture.userCtx, fixture.secrets, secretID) })

	allowRule := createEgressRule(t, fixture.userCtx, fixture.egress, fixture.organizationID, secretID, egressv1.EgressRuleAction_EGRESS_RULE_ACTION_ALLOW)
	allowRuleID := allowRule.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressRule(t, fixture.userCtx, fixture.egress, allowRuleID) })

	attachment := createEgressRuleAttachment(t, fixture.userCtx, fixture.egress, allowRuleID, fixture.agentID)
	attachmentID := attachment.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressRuleAttachment(t, fixture.userCtx, fixture.egress, attachmentID) })

	waitForEgressRuleByAgent(t, ctx, fixture.egress, fixture.agentID, allowRuleID)
	assertSecretDeleteBlockedByEgressRule(t, fixture.userCtx, fixture.secrets, secretID)
	assertSecretResolution(t, fixture.userCtx, fixture.secrets, secretID, egressExpectedSecretValue)
	assertEgressRuleConfiguration(t, allowRule, secretID)
	assertEgressAttachmentListed(t, fixture.userCtx, fixture.egress, fixture.organizationID, fixture.agentID, attachmentID)
}

func TestEgressGatewayDenyAndNoRulePaths(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), egressGatewayTestTimeout)
	t.Cleanup(cancel)

	fixture := setupEgressFixture(t, ctx)
	secret := createEgressSecret(t, fixture.userCtx, fixture.secrets, fixture.organizationID)
	secretID := secret.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressSecret(t, fixture.userCtx, fixture.secrets, secretID) })

	noRuleResp, err := fixture.egress.ListEgressRulesByAgent(ctx, &egressv1.ListEgressRulesByAgentRequest{AgentId: fixture.agentID})
	if err != nil {
		t.Fatalf("list egress rules by agent without attachment: %v", err)
	}
	if len(noRuleResp.GetEgressRules()) != 0 {
		t.Fatalf("expected no egress rules for unattached agent, got %d", len(noRuleResp.GetEgressRules()))
	}

	denyRule := createEgressRule(t, fixture.userCtx, fixture.egress, fixture.organizationID, secretID, egressv1.EgressRuleAction_EGRESS_RULE_ACTION_DENY)
	denyRuleID := denyRule.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressRule(t, fixture.userCtx, fixture.egress, denyRuleID) })

	attachment := createEgressRuleAttachment(t, fixture.userCtx, fixture.egress, denyRuleID, fixture.agentID)
	attachmentID := attachment.GetMeta().GetId()
	t.Cleanup(func() { deleteEgressRuleAttachment(t, fixture.userCtx, fixture.egress, attachmentID) })

	listedRule := waitForEgressRuleByAgent(t, ctx, fixture.egress, fixture.agentID, denyRuleID)
	if listedRule.GetEffect().GetAction() != egressv1.EgressRuleAction_EGRESS_RULE_ACTION_DENY {
		t.Fatalf("expected deny rule for agent, got %s", listedRule.GetEffect().GetAction())
	}
}

type egressFixture struct {
	userCtx        context.Context
	identityID     string
	organizationID string
	agentID        string
	egress         egressv1.EgressRulesServiceClient
	secrets        secretsv1.SecretsServiceClient
}

func setupEgressFixture(t *testing.T, ctx context.Context) egressFixture {
	t.Helper()
	egressConn := dialGRPC(t, envOrDefault("EGRESS_ADDRESS", defaultEgressAddr))
	secretsConn := dialGRPC(t, secretsAddr)
	usersConn := dialGRPC(t, usersAddr)
	orgsConn := dialGRPC(t, orgsAddr)

	egressClient := egressv1.NewEgressRulesServiceClient(egressConn)
	secretsClient := secretsv1.NewSecretsServiceClient(secretsConn)
	usersClient := usersv1.NewUsersServiceClient(usersConn)
	orgsClient := organizationsv1.NewOrganizationsServiceClient(orgsConn)
	authzClient := newAuthorizationClient(t)

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	userCtx := withIdentity(ctx, identityID)
	organizationID := createTestOrganization(t, ctx, orgsClient, identityID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := orgsClient.DeleteOrganization(withIdentity(cleanupCtx, identityID), &organizationsv1.DeleteOrganizationRequest{Id: organizationID}); err != nil {
			t.Logf("cleanup: delete organization %s: %v", organizationID, err)
		}
	})

	agentID := uuid.NewString()
	ensureEgressAgentAuthorization(t, ctx, authzClient, identityID, organizationID, agentID)

	return egressFixture{
		userCtx:        userCtx,
		identityID:     identityID,
		organizationID: organizationID,
		agentID:        agentID,
		egress:         egressClient,
		secrets:        secretsClient,
	}
}

func ensureEgressAgentAuthorization(t *testing.T, ctx context.Context, client authorizationv1.AuthorizationServiceClient, identityID, organizationID, agentID string) {
	t.Helper()
	ensureClusterAdmin(t, ctx, client)
	tuples := []*authorizationv1.TupleKey{
		{User: authorizationIdentityPrefix + identityID, Relation: "can_edit_config", Object: "agent:" + agentID},
		{User: authorizationIdentityPrefix + identityID, Relation: "can_read_config", Object: "agent:" + agentID},
		{User: authorizationOrganizationPrefix + organizationID, Relation: "org", Object: "agent:" + agentID},
	}
	adminCtx := adminContext(ctx)
	if _, err := client.Write(adminCtx, &authorizationv1.WriteRequest{Writes: tuples}); err != nil {
		t.Fatalf("authorization write failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := client.Write(adminContext(cleanupCtx), &authorizationv1.WriteRequest{Deletes: tuples}); err != nil {
			t.Logf("cleanup: authorization delete failed for egress fixture: %v", err)
		}
	})
}

func createEgressSecret(t *testing.T, ctx context.Context, client secretsv1.SecretsServiceClient, organizationID string) *secretsv1.Secret {
	t.Helper()
	return createEgressSecretWithValue(t, ctx, client, organizationID, egressExpectedSecretValue)
}

func createEgressSecretWithValue(t *testing.T, ctx context.Context, client secretsv1.SecretsServiceClient, organizationID, value string) *secretsv1.Secret {
	t.Helper()
	resp, err := client.CreateSecret(ctx, &secretsv1.CreateSecretRequest{
		Title:          "e2e-egress-secret-" + uuid.NewString(),
		Description:    "E2E Egress Gateway secret",
		OrganizationId: organizationID,
		Value:          value,
	})
	if err != nil {
		t.Fatalf("create egress secret: %v", err)
	}
	secret := resp.GetSecret()
	if secret == nil || secret.GetMeta() == nil || secret.GetMeta().GetId() == "" {
		t.Fatal("create egress secret: missing id")
	}
	return secret
}

func deleteEgressSecret(t *testing.T, ctx context.Context, client secretsv1.SecretsServiceClient, secretID string) {
	t.Helper()
	_, err := client.DeleteSecret(ctx, &secretsv1.DeleteSecretRequest{Id: secretID})
	if err != nil && status.Code(err) != codes.NotFound {
		t.Logf("cleanup: delete egress secret %s: %v", secretID, err)
	}
}

func createEgressRule(t *testing.T, ctx context.Context, client egressv1.EgressRulesServiceClient, organizationID, secretID string, action egressv1.EgressRuleAction) *egressv1.EgressRule {
	t.Helper()
	resp, err := client.CreateEgressRule(ctx, &egressv1.CreateEgressRuleRequest{
		OrganizationId: organizationID,
		Name:           fmt.Sprintf("e2e-egress-%s-%s", strings.ToLower(strings.TrimPrefix(action.String(), "EGRESS_RULE_ACTION_")), uuid.NewString()),
		Description:    "E2E Egress Gateway rule",
		Matcher: &egressv1.EgressRuleMatcher{
			DomainPattern: fmt.Sprintf("e2e-egress-%s.example.com", uuid.NewString()),
			Ports:         []int32{443},
			Methods:       []string{"GET"},
			PathPattern:   "/anything/*",
		},
		Effect: &egressv1.EgressRuleEffect{
			Action: action.Enum(),
			Inject: []*egressv1.EgressRuleHeader{{
				Name:       "Authorization",
				Scheme:     egressv1.HeaderAuthScheme_HEADER_AUTH_SCHEME_BEARER,
				Credential: &egressv1.EgressRuleHeader_SecretId{SecretId: secretID},
			}},
		},
	})
	if err != nil {
		t.Fatalf("create egress rule: %v", err)
	}
	rule := resp.GetEgressRule()
	if rule == nil || rule.GetMeta() == nil || rule.GetMeta().GetId() == "" {
		t.Fatal("create egress rule: missing id")
	}
	return rule
}

func deleteEgressRule(t *testing.T, ctx context.Context, client egressv1.EgressRulesServiceClient, ruleID string) {
	t.Helper()
	_, err := client.DeleteEgressRule(ctx, &egressv1.DeleteEgressRuleRequest{Id: ruleID})
	if err != nil && status.Code(err) != codes.NotFound {
		t.Logf("cleanup: delete egress rule %s: %v", ruleID, err)
	}
}

func createEgressRuleAttachment(t *testing.T, ctx context.Context, client egressv1.EgressRulesServiceClient, ruleID, agentID string) *egressv1.EgressRuleAttachment {
	t.Helper()
	resp, err := client.CreateEgressRuleAttachment(ctx, &egressv1.CreateEgressRuleAttachmentRequest{RuleId: ruleID, AgentId: agentID})
	if err != nil {
		t.Fatalf("create egress rule attachment: %v", err)
	}
	attachment := resp.GetEgressRuleAttachment()
	if attachment == nil || attachment.GetMeta() == nil || attachment.GetMeta().GetId() == "" {
		t.Fatal("create egress rule attachment: missing id")
	}
	return attachment
}

func deleteEgressRuleAttachment(t *testing.T, ctx context.Context, client egressv1.EgressRulesServiceClient, attachmentID string) {
	t.Helper()
	_, err := client.DeleteEgressRuleAttachment(ctx, &egressv1.DeleteEgressRuleAttachmentRequest{Id: attachmentID})
	if err != nil && status.Code(err) != codes.NotFound {
		t.Logf("cleanup: delete egress rule attachment %s: %v", attachmentID, err)
	}
}

func waitForEgressRuleByAgent(t *testing.T, ctx context.Context, client egressv1.EgressRulesServiceClient, agentID, ruleID string) *egressv1.EgressRule {
	t.Helper()
	var matched *egressv1.EgressRule
	pollCtx, pollCancel := context.WithTimeout(ctx, egressRulePropagationTimeout)
	defer pollCancel()
	if err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		resp, err := client.ListEgressRulesByAgent(ctx, &egressv1.ListEgressRulesByAgentRequest{AgentId: agentID})
		if err != nil {
			return err
		}
		for _, rule := range resp.GetEgressRules() {
			if rule.GetMeta().GetId() == ruleID {
				matched = rule
				return nil
			}
		}
		return fmt.Errorf("egress rule %s not listed for agent %s", ruleID, agentID)
	}); err != nil {
		t.Fatalf("wait for egress rule by agent: %v", err)
	}
	return matched
}

func assertSecretDeleteBlockedByEgressRule(t *testing.T, ctx context.Context, client secretsv1.SecretsServiceClient, secretID string) {
	t.Helper()
	_, err := client.DeleteSecret(ctx, &secretsv1.DeleteSecretRequest{Id: secretID})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition deleting referenced secret, got %v", err)
	}
}

func assertSecretResolution(t *testing.T, ctx context.Context, client secretsv1.SecretsServiceClient, secretID, expected string) {
	t.Helper()
	resp, err := client.ResolveSecret(ctx, &secretsv1.ResolveSecretRequest{Id: secretID})
	if err != nil {
		t.Fatalf("resolve egress secret: %v", err)
	}
	if resp.GetValue() != expected {
		t.Fatalf("resolved secret mismatch: got %q want %q", resp.GetValue(), expected)
	}
}

func assertEgressRuleConfiguration(t *testing.T, rule *egressv1.EgressRule, secretID string) {
	t.Helper()
	matcher := rule.GetMatcher()
	if matcher == nil {
		t.Fatal("egress rule matcher missing")
	}
	if matcher.GetDomainPattern() == "" || len(matcher.GetPorts()) != 1 || matcher.GetPorts()[0] != 443 {
		t.Fatalf("unexpected matcher: %+v", matcher)
	}
	effect := rule.GetEffect()
	if effect == nil || effect.GetAction() != egressv1.EgressRuleAction_EGRESS_RULE_ACTION_ALLOW {
		t.Fatalf("unexpected effect action: %+v", effect)
	}
	if len(effect.GetInject()) != 1 {
		t.Fatalf("expected 1 injected header, got %d", len(effect.GetInject()))
	}
	header := effect.GetInject()[0]
	if header.GetName() != "Authorization" || header.GetScheme() != egressv1.HeaderAuthScheme_HEADER_AUTH_SCHEME_BEARER || header.GetSecretId() != secretID {
		t.Fatalf("unexpected injected header: %+v", header)
	}
}

func assertEgressAttachmentListed(t *testing.T, ctx context.Context, client egressv1.EgressRulesServiceClient, organizationID, agentID, attachmentID string) {
	t.Helper()
	resp, err := client.ListEgressRuleAttachments(ctx, &egressv1.ListEgressRuleAttachmentsRequest{
		OrganizationId: organizationID,
		AgentId:        &agentID,
	})
	if err != nil {
		t.Fatalf("list egress rule attachments by agent: %v", err)
	}
	for _, attachment := range resp.GetEgressRuleAttachments() {
		if attachment.GetMeta().GetId() == attachmentID {
			return
		}
	}
	t.Fatalf("egress rule attachment %s not listed for agent %s", attachmentID, agentID)
}
