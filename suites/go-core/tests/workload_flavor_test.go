//go:build e2e && (svc_agents_orchestrator || svc_images || smoke)

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	imagesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/images/v1"
	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// A flavor is the one thing a workload asks for when it wants a size, and the
// runner is what turns that name into requests and limits. Before this was
// wired the platform resolved a flavor, recorded it, and billed by it, while
// every pod still came out BestEffort -- the name reached the record and never
// the container. This asserts the whole path: an environment names a flavor,
// the Orchestrator sends the name, and the runner sizes the pod with it.
func TestWorkloadIsSizedByItsFlavor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = asOwner(t, ctx)

	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	orgID := gatewayOrganizationID(t)
	runnerID := catalogRunnerID(t, ctx)

	flavor := pickCatalogFlavor(t, ctx, runnerID)
	t.Logf("sizing against flavor %q", flavor.GetName())

	workspace := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_WORKSPACE)
	workspaceTag := discoveredTag(t, ctx, workspace.GetMeta().GetId())

	environment := createEnvironment(t, ctx, agentsClient, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:    orgID,
		Name:              fmt.Sprintf("e2e-flavor-%s", uuid.NewString()[:8]),
		RunnerId:          runnerID,
		Flavor:            flavor.GetName(),
		WorkspaceImageId:  workspace.GetMeta().GetId(),
		WorkspaceImageTag: workspaceTag,
	})
	environmentID := environment.GetMeta().GetId()

	sandbox := createSandbox(t, ctx, agentsClient, orgID, environmentID)
	pod := waitForWorkloadPod(t, ctx, sandbox)

	if len(pod.Spec.Containers) == 0 {
		t.Fatal("expected at least the main container")
	}

	main := pod.Spec.Containers[0]
	requireSized(t, "main container "+main.Name, main.Resources)
	assertQuantityMatches(t, "main cpu request", main.Resources.Requests.Cpu().String(), flavor.GetResources().GetRequestsCpu())
	assertQuantityMatches(t, "main memory request", main.Resources.Requests.Memory().String(), flavor.GetResources().GetRequestsMemory())
	assertQuantityMatches(t, "main cpu limit", main.Resources.Limits.Cpu().String(), flavor.GetResources().GetLimitsCpu())
	assertQuantityMatches(t, "main memory limit", main.Resources.Limits.Memory().String(), flavor.GetResources().GetLimitsMemory())

	// Sidecars are sized by the same flavor, from its own sidecar budget --
	// there is no per-MCP size anywhere in the platform, deliberately. A
	// sandbox carries none today, so this asserts on whatever the workload has
	// rather than requiring one.
	for _, sidecar := range pod.Spec.Containers[1:] {
		requireSized(t, "sidecar "+sidecar.Name, sidecar.Resources)
		// The catalog report carries only the workload size, never the sidecar
		// budget behind it -- that is the runner's own business -- so this
		// asserts the sidecar got a budget of its own, not the main one.
		if sidecar.Resources.Limits.Memory().Equal(*main.Resources.Limits.Memory()) {
			t.Errorf("sidecar %s got the main container's budget (%s)", sidecar.Name, sidecar.Resources.Limits.Memory())
		}
	}

	// Init containers are left out on purpose: Kubernetes takes the pod's
	// request as max(sum of containers, max of init containers), so sizing them
	// would make a workload reserve its flavor plus its bootstrap.
	for _, initContainer := range pod.Spec.InitContainers {
		if len(initContainer.Resources.Requests) != 0 || len(initContainer.Resources.Limits) != 0 {
			t.Errorf("init container %s is sized (%v); the flavor must not reach it",
				initContainer.Name, initContainer.Resources)
		}
	}
}

// pickCatalogFlavor prefers an entry that is not the runner's default, so the
// test proves the named flavor was honoured rather than the fallback taken.
func pickCatalogFlavor(t *testing.T, ctx context.Context, runnerID string) *runnersv1.Flavor {
	t.Helper()
	client := runnersv1.NewRunnersServiceClient(dialGRPC(t, runnersAddr))
	listed, err := client.ListFlavors(ctx, &runnersv1.ListFlavorsRequest{
		RunnerId: &runnerID,
		PageSize: 100,
	})
	if err != nil {
		t.Fatalf("ListFlavors: %v", err)
	}
	flavors := listed.GetFlavors()
	if len(flavors) == 0 {
		t.Fatalf("runner %s reports no flavors; nothing can be sized", runnerID)
	}
	for _, flavor := range flavors {
		if !flavor.GetDefault() && flavor.GetResources() != nil {
			return flavor
		}
	}
	return flavors[0]
}

func requireSized(t *testing.T, what string, resources corev1.ResourceRequirements) {
	t.Helper()
	if len(resources.Requests) == 0 || len(resources.Limits) == 0 {
		t.Fatalf("%s is unsized (requests=%v limits=%v); the flavor never reached the container",
			what, resources.Requests, resources.Limits)
	}
}

// assertQuantityMatches compares by parsed value: the catalog may say "2" where
// Kubernetes reports "2", but also "0.5" where it reports "500m".
func assertQuantityMatches(t *testing.T, what, got, want string) {
	t.Helper()
	if want == "" {
		return
	}
	gotQuantity, err := resource.ParseQuantity(got)
	if err != nil {
		t.Fatalf("%s: parse pod value %q: %v", what, got, err)
	}
	wantQuantity, err := resource.ParseQuantity(want)
	if err != nil {
		t.Fatalf("%s: parse catalog value %q: %v", what, want, err)
	}
	if gotQuantity.Cmp(wantQuantity) != 0 {
		t.Errorf("%s: pod has %s, flavor declares %s", what, got, want)
	}
}
