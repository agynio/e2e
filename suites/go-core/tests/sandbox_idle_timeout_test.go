//go:build e2e && (svc_organizations || svc_agents_orchestrator || smoke)

package tests

import (
	"context"

	"testing"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// A fresh organization per test: these move the sandbox settings, and the
// bootstrap organization is shared with every other suite.
func newOrganization(ctx context.Context, t *testing.T, client organizationsv1.OrganizationsServiceClient, ownerID string) string {
	t.Helper()
	created, err := client.CreateOrganization(withIdentity(ctx, ownerID), &organizationsv1.CreateOrganizationRequest{
		Name: "Sandbox Bounds " + uuid.NewString(),
	})
	require.NoError(t, err)
	organizationID := created.GetOrganization().GetId()
	require.NotEmpty(t, organizationID)
	return organizationID
}

// The default is what a creator who names nothing gets; the ceiling is what the
// organization will pay for when someone has thought about it. Collapsing them
// into one field would make the default the most expensive option on offer.
func TestOrganizationSandboxIdleBounds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	client := organizationsv1.NewOrganizationsServiceClient(dialGRPC(t, orgsAddr))
	// A person the platform knows, not a UUID. Authorization resolves the
	// caller's identity before it reaches the rule under test, so a fabricated
	// one is refused with PermissionDenied and the assertion never sees the
	// InvalidArgument it is about.
	ownerID := suiteUserIdentity(t, ctx)
	ownerCtx := withIdentity(ctx, ownerID)
	organizationID := newOrganization(ctx, t, client, ownerID)

	fetched, err := client.GetOrganization(ctx, &organizationsv1.GetOrganizationRequest{Id: organizationID})
	require.NoError(t, err)
	require.Equal(t, "24h", fetched.GetOrganization().GetSandboxMaxIdleTimeout(),
		"a new organization starts at the platform maximum and narrows it deliberately")

	updated, err := client.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
		Id:                    organizationID,
		SandboxMaxIdleTimeout: proto.String("4h"),
	})
	require.NoError(t, err)
	require.Equal(t, "4h0m0s", updated.GetOrganization().GetSandboxMaxIdleTimeout())

	// The pair is checked on whichever half the request names.
	_, err = client.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
		Id:                        organizationID,
		SandboxDefaultIdleTimeout: proto.String("6h"),
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err),
		"a default above the stored ceiling must be refused")

	_, err = client.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
		Id:                    organizationID,
		SandboxMaxIdleTimeout: proto.String("10m"),
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err),
		"a ceiling below the stored default must be refused")

	// Both moving together is consistent even though either alone is not.
	raised, err := client.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
		Id:                        organizationID,
		SandboxDefaultIdleTimeout: proto.String("6h"),
		SandboxMaxIdleTimeout:     proto.String("8h"),
	})
	require.NoError(t, err)
	require.Equal(t, "6h0m0s", raised.GetOrganization().GetSandboxDefaultIdleTimeout())
	require.Equal(t, "8h0m0s", raised.GetOrganization().GetSandboxMaxIdleTimeout())
}

// CreateSandbox reads the organization's bounds rather than a constant, and
// refuses a request above the ceiling naming it rather than clamping — a
// silently reduced timeout is a number the engineer never sees and plans around
// wrongly.
//
// This runs in the bootstrap organization, and for the reason its sibling does:
// a sandbox needs an environment and a fresh organization owns none. The
// request used to name an absent environment on the grounds that the ceiling is
// settled first, which stopped being true -- the caller is authorized against
// the environment before the bound is read, so the probe came back
// PermissionDenied and the refusal under test never happened. The ceiling is
// perturbed and restored, so the window is narrow and this suite is sequential.
func TestCreateSandboxHonoursTheOrganizationCeiling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	organizations := organizationsv1.NewOrganizationsServiceClient(dialGRPC(t, orgsAddr))
	agents := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))

	organizationID := gatewayOrganizationID(t)
	ownerCtx := withIdentity(ctx, fetchGatewayIdentity(t, gatewayAPIToken(t)).IdentityID)

	environmentID := suiteEnvironment(t, ownerCtx, agents, organizationID, codexRuntime)

	before, err := organizations.GetOrganization(ctx, &organizationsv1.GetOrganizationRequest{Id: organizationID})
	require.NoError(t, err)
	originalCeiling := before.GetOrganization().GetSandboxMaxIdleTimeout()
	t.Cleanup(func() {
		_, _ = organizations.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
			Id:                    organizationID,
			SandboxMaxIdleTimeout: proto.String(originalCeiling),
		})
	})

	_, err = organizations.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
		Id:                    organizationID,
		SandboxMaxIdleTimeout: proto.String("2h"),
	})
	require.NoError(t, err)

	err = requestSandbox(ctx, agents, ownerCtx, organizationID, environmentID, "6h")
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "2h", "the refusal must name the ceiling it exceeded")

	// At the ceiling exactly, the bound is passed. Whatever happens next is not
	// this test's business, so long as it is not the bound refusing it.
	err = requestSandbox(ctx, agents, ownerCtx, organizationID, environmentID, "2h")
	if err != nil {
		require.NotContains(t, err.Error(), "sandbox_max_idle_timeout",
			"a request at the ceiling must pass the bound")
	}

	// Raising the ceiling makes the same request acceptable, which is what
	// shows the value is read from the organization rather than compiled in.
	_, err = organizations.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
		Id:                    organizationID,
		SandboxMaxIdleTimeout: proto.String("8h"),
	})
	require.NoError(t, err)

	err = requestSandbox(ctx, agents, ownerCtx, organizationID, environmentID, "6h")
	if err != nil {
		require.NotContains(t, err.Error(), "sandbox_max_idle_timeout",
			"raising the ceiling must admit a request it previously refused")
	}
}

// Bounds are snapshotted at creation. Lowering them afterwards is what an
// organization does when it changes its mind, and it must not reach back into
// sandboxes already running on the old numbers.
//
// This runs in the bootstrap organization because the ceiling under test is one
// of its settings. The environment is the suite's own: taking whichever the
// organization listed first meant taking one another test had created and this
// caller has no grant on, and the probe then came back PermissionDenied instead
// of the refusal under test. The settings are perturbed and restored, so the
// window is narrow and this suite is sequential.
func TestSandboxBoundsAreNotReReadAfterCreation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	organizations := organizationsv1.NewOrganizationsServiceClient(dialGRPC(t, orgsAddr))
	agents := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))

	organizationID := gatewayOrganizationID(t)
	ownerCtx := withIdentity(ctx, fetchGatewayIdentity(t, gatewayAPIToken(t)).IdentityID)

	environmentID := suiteEnvironment(t, ownerCtx, agents, organizationID, codexRuntime)

	before, err := organizations.GetOrganization(ctx, &organizationsv1.GetOrganizationRequest{Id: organizationID})
	require.NoError(t, err)
	original := before.GetOrganization()
	t.Cleanup(func() {
		_, _ = organizations.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
			Id:                        organizationID,
			SandboxDefaultIdleTimeout: proto.String(original.GetSandboxDefaultIdleTimeout()),
			SandboxMaxIdleTimeout:     proto.String(original.GetSandboxMaxIdleTimeout()),
		})
	})

	created, err := agents.CreateSandbox(ownerCtx, &agentsv1.CreateSandboxRequest{
		OrganizationId: organizationID,
		EnvironmentId:  environmentID,
	})
	require.NoError(t, err)
	sandbox := created.GetSandbox()
	t.Cleanup(func() {
		_, _ = agents.DeleteSandbox(ownerCtx, &agentsv1.DeleteSandboxRequest{Id: sandbox.GetMeta().GetId()})
	})
	recorded := sandbox.GetIdleTimeout()
	require.NotEmpty(t, recorded)
	require.Equal(t, original.GetSandboxDefaultIdleTimeout(), recorded,
		"a sandbox naming no timeout records the organization's default")

	_, err = organizations.UpdateOrganization(ownerCtx, &organizationsv1.UpdateOrganizationRequest{
		Id:                        organizationID,
		SandboxDefaultIdleTimeout: proto.String("5m"),
		SandboxMaxIdleTimeout:     proto.String("10m"),
	})
	require.NoError(t, err)

	fetched, err := agents.GetSandbox(ownerCtx, &agentsv1.GetSandboxRequest{
		Ref: &agentsv1.GetSandboxRequest_Id{Id: sandbox.GetMeta().GetId()},
	})
	require.NoError(t, err)
	require.Equal(t, recorded, fetched.GetSandbox().GetIdleTimeout(),
		"lowering the organization's settings must leave a live sandbox on the value it started with")
}

func requestSandbox(ctx context.Context, agents agentsv1.AgentsServiceClient, ownerCtx context.Context, organizationID, environmentID, idleTimeout string) error {
	_, err := agents.CreateSandbox(ownerCtx, &agentsv1.CreateSandboxRequest{
		OrganizationId: organizationID,
		EnvironmentId:  environmentID,
		IdleTimeout:    proto.String(idleTimeout),
	})
	return err
}
