//go:build e2e && (svc_gateway || svc_agents_orchestrator || svc_runners || svc_metering || svc_k8s_runner || svc_organizations || svc_llm || svc_llm_proxy || svc_images || smoke)

package tests

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	imagesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/images/v1"
	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The fixtures every suite that touches an agent or a sandbox shares: the
// catalog records an image reference resolves through, the runner an
// environment is placed on, and the environment itself.
//
// They live here rather than beside the catalog tests because an agent cannot
// be created without them. An agent takes its CLI from its environment's agent
// runtime image, and the Orchestrator refuses to assemble a workload that has
// none -- so every suite that creates an agent needs the whole chain.

const (
	// The agent runtimes the release publishes to the image catalog. The suite
	// names one and resolves it there, rather than being handed a registry
	// reference and a tag: the platform already discovers both, and a caller
	// supplying them is a second source of truth that goes stale on its own.
	codexRuntime  = "codex"
	agnRuntime    = "agn"
	claudeRuntime = "claude"
)

var (
	agentsAddr  = envOrDefault("AGENTS_ADDRESS", "agents:50051")
	imagesAddr  = envOrDefault("IMAGES_ADDRESS", "images:50051")
	runnersAddr = envOrDefault("RUNNERS_ADDRESS", "runners:50051")
)

// Authoring an image is an owner's write, so every call here carries the
// identity the Gateway would attach. Without it the service refuses, which is
// the point of requiring one.
func asOwner(t *testing.T, ctx context.Context) context.Context {
	t.Helper()
	return withIdentity(ctx, fetchGatewayIdentity(t, gatewayAPIToken(t)).IdentityID)
}

func newImagesClient(t *testing.T) imagesv1.ImagesServiceClient {
	t.Helper()
	return imagesv1.NewImagesServiceClient(dialGRPC(t, imagesAddr))
}

// registerCatalogImage registers a repository the platform itself publishes, so
// discovery has real tags to find rather than a stub's.
func registerCatalogImage(t *testing.T, ctx context.Context, organizationID string, imageType imagesv1.ImageType) *imagesv1.Image {
	t.Helper()
	return registerCatalogImageFrom(t, ctx, organizationID, imageType,
		envOrDefault("TEST_PUBLIC_REPOSITORY", "ghcr.io/agynio/devcontainer-go"))
}

func registerCatalogImageFrom(t *testing.T, ctx context.Context, organizationID string, imageType imagesv1.ImageType, repository string) *imagesv1.Image {
	t.Helper()
	client := newImagesClient(t)
	created, err := client.CreateImage(ctx, &imagesv1.CreateImageRequest{
		OrganizationId: organizationID,
		Name:           fmt.Sprintf("e2e-catalog-%s", uuid.NewString()[:8]),
		Type:           imageType,
		Repository:     repository,
		Visibility:     imagesv1.ImageVisibility_IMAGE_VISIBILITY_INTERNAL,
	})
	if err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	image := created.GetImage()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := client.DeleteImage(asOwner(t, cleanupCtx), &imagesv1.DeleteImageRequest{Id: image.GetMeta().GetId()}); err != nil {
			t.Logf("cleanup DeleteImage: %v", err)
		}
	})
	return image
}

// discoveredTag returns a tag the catalog has actually seen. Registration kicks
// off discovery in the background, so this refreshes rather than assuming.
func discoveredTag(t *testing.T, ctx context.Context, imageID string) string {
	t.Helper()
	refreshed, err := newImagesClient(t).RefreshImage(ctx, &imagesv1.RefreshImageRequest{ImageId: imageID})
	if err != nil {
		t.Fatalf("RefreshImage: %v", err)
	}
	preferred := envOrDefault("TEST_PUBLIC_TAG", "1.2.0")
	for _, version := range refreshed.GetVersions() {
		if version.GetTag() == preferred {
			return preferred
		}
	}
	if len(refreshed.GetVersions()) == 0 {
		t.Fatal("discovery found no versions")
	}
	return refreshed.GetVersions()[0].GetTag()
}

func createEnvironment(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, request *agentsv1.CreateEnvironmentRequest) *agentsv1.Environment {
	t.Helper()
	// An environment has to say who can reach it, and the callers here do not
	// care -- they are about images and sandboxes. Left unset it is UNSPECIFIED,
	// which agents refuses with "availability: must be internal or private", so
	// the narrower of the two stands in.
	if request.GetAvailability() == agentsv1.EnvironmentAvailability_ENVIRONMENT_AVAILABILITY_UNSPECIFIED {
		request.Availability = agentsv1.EnvironmentAvailability_ENVIRONMENT_AVAILABILITY_PRIVATE
	}
	created, err := client.CreateEnvironment(ctx, request)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	environment := created.GetEnvironment()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := client.DeleteEnvironment(asOwner(t, cleanupCtx), &agentsv1.DeleteEnvironmentRequest{Id: environment.GetMeta().GetId()}); err != nil {
			if status.Code(err) == codes.FailedPrecondition {
				// A terminated sandbox is collected on a later cycle and still
				// references the environment until then.
				t.Logf("cleanup: environment %s outlives its sandbox; it is collected later", environment.GetMeta().GetId())
			} else {
				t.Logf("cleanup DeleteEnvironment: %v", err)
			}
		}
	})
	return environment
}

// catalogRunnerID picks the organization's runner. An environment must name one,
// and the local platform provisions exactly one.
func catalogRunnerID(t *testing.T, ctx context.Context) string {
	t.Helper()
	organizationID := gatewayOrganizationID(t)
	listed, err := runnersv1.NewRunnersServiceClient(dialGRPC(t, runnersAddr)).ListRunners(ctx, &runnersv1.ListRunnersRequest{
		OrganizationId: &organizationID,
		PageSize:       100,
	})
	if err != nil {
		t.Fatalf("ListRunners: %v", err)
	}
	for _, runner := range listed.GetRunners() {
		if id := runner.GetMeta().GetId(); id != "" {
			return id
		}
	}
	t.Fatal("no runner in the organization")
	return ""
}

// suiteEnvironment builds the environment an agent runs in, on the agent
// runtime the caller names.
//
// The runtime is a catalog image the release ships -- codex, agn, claude, all
// public and all discovered -- rather than a registry reference the caller
// supplies. An environment names a catalog record, so a reference would have to
// be registered as one first; and the release already publishes exactly these
// three, so registering a second copy of one is a fixture pretending to be a
// platform.
// Shared across the run, keyed on the runtime, because an environment describes
// how a workload runs rather than anything about one test. Building one per
// agent left a platform carrying an environment per test per run -- ninety-six
// of them on a local VM after a day -- which is both litter and load: the
// Agents service walks every environment on startup, and enough of them stop it
// coming up at all.
//
// Deliberately never deleted. It is created once, outlives the test that first
// asked for it, and a run's worth of them costs one row per runtime.
var (
	suiteEnvironmentsMu sync.Mutex
	suiteEnvironments   = map[string]string{}
)

func suiteEnvironment(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, organizationID, runtime string) string {
	t.Helper()
	suiteEnvironmentsMu.Lock()
	defer suiteEnvironmentsMu.Unlock()
	key := organizationID + "/" + runtime
	if id, ok := suiteEnvironments[key]; ok {
		return id
	}
	// The workload's main container. The free-form field rather than a catalog
	// workspace image: the suites need something the agyn binary can run in,
	// and nothing about the image under test.
	id := newSuiteEnvironment(t, ctx, client, organizationID, runtime, "alpine:3.21")
	suiteEnvironments[key] = id
	return id
}

// newSuiteEnvironment builds one outright, with the main container named. The
// shared fixture above calls it once per runtime; a test about an image that
// cannot be pulled calls it directly, because its whole point is an image the
// others must not get and it repairs that image in place.
func newSuiteEnvironment(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, organizationID, runtime, workspaceImage string) string {
	t.Helper()
	imageID, tag := platformAgentRuntime(t, ctx, organizationID, runtime)
	environment := createEnvironment(t, ctx, client, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:       organizationID,
		Name:                 fmt.Sprintf("e2e-env-%s", uuid.NewString()[:8]),
		RunnerId:             catalogRunnerID(t, ctx),
		Image:                workspaceImage,
		AgentRuntimeImageId:  imageID,
		AgentRuntimeImageTag: tag,
	})
	return environment.GetMeta().GetId()
}

// platformAgentRuntime finds one of the agent runtimes the release publishes,
// and the newest release tag discovery has seen for it.
func platformAgentRuntime(t *testing.T, ctx context.Context, organizationID, name string) (string, string) {
	t.Helper()
	client := newImagesClient(t)
	listed, err := client.ListImages(ctx, &imagesv1.ListImagesRequest{
		OrganizationId: organizationID,
		Type:           imagesv1.ImageType_IMAGE_TYPE_AGENT_RUNTIME,
		PageSize:       100,
	})
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	var imageID string
	seen := []string{}
	for _, image := range listed.GetImages() {
		seen = append(seen, image.GetName())
		if image.GetName() == name {
			imageID = image.GetMeta().GetId()
			break
		}
	}
	if imageID == "" {
		t.Fatalf("no agent runtime image named %q in the catalog; found %v", name, seen)
	}

	versions, err := client.ListVersions(ctx, &imagesv1.ListVersionsRequest{
		ImageId:  imageID,
		PageSize: 200,
	})
	if err != nil {
		t.Fatalf("ListVersions for %s: %v", name, err)
	}
	tag := newestReleaseTag(versions.GetVersions())
	if tag == "" {
		t.Fatalf("no released tag discovered for agent runtime %q", name)
	}
	return imageID, tag
}

// newestReleaseTag picks the highest x.y.z among the discovered tags.
//
// Only fully-qualified ones: a repository also carries floating tags -- latest,
// 0, 0.5 -- and a sha-<commit> per build, none of which name a release. Pinning
// a suite to one of those is how a run silently moves under itself.
func newestReleaseTag(versions []*imagesv1.ImageVersion) string {
	best := ""
	var bestParts []int
	for _, version := range versions {
		parts, ok := semverParts(version.GetTag())
		if !ok {
			continue
		}
		if bestParts == nil || compareSemver(parts, bestParts) > 0 {
			best, bestParts = version.GetTag(), parts
		}
	}
	return best
}

func semverParts(tag string) ([]int, bool) {
	fields := strings.Split(tag, ".")
	if len(fields) != 3 {
		return nil, false
	}
	parts := make([]int, 3)
	for i, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil {
			return nil, false
		}
		parts[i] = value
	}
	return parts, true
}

func compareSemver(a, b []int) int {
	for i := range a {
		switch {
		case a[i] > b[i]:
			return 1
		case a[i] < b[i]:
			return -1
		}
	}
	return 0
}
