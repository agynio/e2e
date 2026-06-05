//go:build e2e && svc_egress && !(svc_agents_orchestrator || svc_llm || svc_llm_proxy || smoke)

package tests

import (
	"context"
	"fmt"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

func resolveOrCreateUser(t *testing.T, ctx context.Context, client usersv1.UsersServiceClient) string {
	t.Helper()
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	subject := fmt.Sprintf("e2e-egress-%s", uuid.NewString())
	name := fmt.Sprintf("E2E Egress %s", subject)
	email := fmt.Sprintf("%s@test.local", subject)

	resp, err := client.ResolveOrCreateUser(callCtx, &usersv1.ResolveOrCreateUserRequest{
		OidcSubject: subject,
		Name:        name,
		Email:       email,
	})
	if err != nil {
		t.Fatalf("resolve user %s: %v", subject, err)
	}
	if resp == nil || resp.GetUser() == nil || resp.GetUser().GetMeta() == nil {
		t.Fatalf("resolve user %s: missing user metadata", subject)
	}
	identityID := strings.TrimSpace(resp.GetUser().GetMeta().GetId())
	if identityID == "" {
		t.Fatalf("resolve user %s: identity id missing", subject)
	}
	return identityID
}

func createTestOrganization(t *testing.T, ctx context.Context, client organizationsv1.OrganizationsServiceClient, identityID string) string {
	t.Helper()
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	name := fmt.Sprintf("e2e-egress-org-%s", uuid.NewString())
	callCtx = withIdentity(callCtx, identityID)
	resp, err := client.CreateOrganization(callCtx, &organizationsv1.CreateOrganizationRequest{Name: name})
	if err != nil {
		t.Fatalf("create organization %s: %v", name, err)
	}
	if resp == nil || resp.GetOrganization() == nil {
		t.Fatalf("create organization %s: missing organization", name)
	}
	orgID := strings.TrimSpace(resp.GetOrganization().GetId())
	if orgID == "" {
		t.Fatalf("create organization %s: id missing", name)
	}
	return orgID
}
