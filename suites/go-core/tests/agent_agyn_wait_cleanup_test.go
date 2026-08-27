//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
)

// This test needs one exact nickname, because the scripted turn addresses agent
// B by handle: `--add @e2e-agyn-wait-b-fixed`. A nickname is claimed once per
// organization, and the suite runs every time against the same organization, so
// the name is a resource the test borrows rather than one it invents.
//
// That makes cleanup load-bearing. When it does not run, the leftover agent
// holds the handle and every later run fails in setup with AlreadyExists --
// one failed run wedges the test for good.
const agynWaitCleanupTimeout = 30 * time.Second

// cleanupContext is a fresh deadline for the teardown of a test that has
// already spent its own.
//
// A t.Cleanup that reuses the test context inherits its deadline, which by then
// is usually blown -- so the teardown that frees the handle fails first, with
// DeadlineExceeded, exactly when the test failed and cleanup matters most.
func cleanupContext(t *testing.T, identityID string) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), agynWaitCleanupTimeout)
	return withIdentity(ctx, identityID), cancel
}

// releaseNickname hands the handle back before claiming it.
//
// Recovery belongs here rather than in a runbook: the leftover sits in the same
// organization this test already acts in, so the test can see it and delete it.
func releaseNickname(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, organizationID, nickname string) {
	t.Helper()
	pageToken := ""
	for {
		resp, err := client.ListAgents(ctx, &agentsv1.ListAgentsRequest{
			OrganizationId: organizationID,
			PageSize:       100,
			PageToken:      pageToken,
		})
		if err != nil {
			t.Logf("release nickname %q: list agents: %v", nickname, err)
			return
		}
		for _, agent := range resp.GetAgents() {
			if agent.GetNickname() != nickname {
				continue
			}
			id := agent.GetMeta().GetId()
			t.Logf("release nickname %q: deleting leftover agent %s from an earlier run", nickname, id)
			cleanupAgentEnvs(t, ctx, client, organizationID, id)
			deleteAgent(t, ctx, client, organizationID, id)
		}
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			return
		}
	}
}
