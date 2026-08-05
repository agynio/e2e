//go:build e2e && (svc_agents_orchestrator || svc_images || smoke)

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	imagesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/images/v1"
	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Covers the path an image reference takes end to end: registered in the
// catalog, named by an environment, rewritten by the Orchestrator onto the
// image proxy, and pulled by the runner with a credential minted for that one
// workload. Every step of it used to be a free-form string typed by an operator.

var imagesAddr = envOrDefault("IMAGES_ADDRESS", "images:50051")

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
	client := newImagesClient(t)
	created, err := client.CreateImage(ctx, &imagesv1.CreateImageRequest{
		OrganizationId: organizationID,
		Name:           fmt.Sprintf("e2e-catalog-%s", uuid.NewString()[:8]),
		Type:           imageType,
		Repository:     envOrDefault("TEST_PUBLIC_REPOSITORY", "ghcr.io/agynio/devcontainer-go"),
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

// An environment that names catalog records makes the Orchestrator rewrite the
// reference onto the proxy and attach a credential. Nothing in the pod may name
// the upstream registry.
func TestAWorkloadPullsThroughTheProxy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = asOwner(t, ctx)

	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	orgID := gatewayOrganizationID(t)

	workspace := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_WORKSPACE)
	runtime := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_AGENT_RUNTIME)
	workspaceTag := discoveredTag(t, ctx, workspace.GetMeta().GetId())
	runtimeTag := discoveredTag(t, ctx, runtime.GetMeta().GetId())

	environment := createEnvironment(t, ctx, agentsClient, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:       orgID,
		Name:                 fmt.Sprintf("e2e-catalog-%s", uuid.NewString()[:8]),
		RunnerId:             catalogRunnerID(t, ctx),
		WorkspaceImageId:     workspace.GetMeta().GetId(),
		WorkspaceImageTag:    workspaceTag,
		AgentRuntimeImageId:  runtime.GetMeta().GetId(),
		AgentRuntimeImageTag: runtimeTag,
	})

	sandbox := createSandbox(t, ctx, agentsClient, orgID, environment.GetMeta().GetId())
	pod := waitForWorkloadPod(t, ctx, agentsClient, sandbox)

	proxyHost := envOrDefault("IMAGE_PROXY_HOST", "registry.agyn.dev")
	wantPrefix := fmt.Sprintf("%s/", proxyHost)

	var main string
	for _, container := range pod.Spec.Containers {
		if strings.HasPrefix(container.Image, wantPrefix) {
			main = container.Image
		}
	}
	if main == "" {
		var seen []string
		for _, container := range pod.Spec.Containers {
			seen = append(seen, container.Image)
		}
		t.Fatalf("no container pulls through %s; images were %v", proxyHost, seen)
	}
	// The reference carries no port: production serves the proxy on 443, and
	// containerd keys its store by the reference it pulled with, so a port here
	// would make a locally baked layer a different image than the same content
	// in production.
	if strings.Contains(strings.TrimPrefix(main, wantPrefix), ":") {
		host, _, _ := strings.Cut(main, "/")
		if strings.Contains(host, ":") {
			t.Fatalf("image %q names a port; the reference must match production", main)
		}
	}
	if !strings.HasSuffix(main, ":"+workspaceTag) {
		t.Fatalf("image %q does not carry the tag the environment named (%s)", main, workspaceTag)
	}

	// Without the credential the pull is refused: the proxy serves nothing
	// anonymously.
	if len(pod.Spec.ImagePullSecrets) == 0 {
		t.Fatal("workload has no image pull secret")
	}
}

// An environment with a workspace image and no agent runtime is workspace-only:
// a sandbox can use it, an agent has no CLI to run. Refusing at create time
// beats a workload that starts and does nothing.
func TestAnAgentNeedsAnAgentRuntimeImage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ctx = asOwner(t, ctx)

	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	orgID := gatewayOrganizationID(t)

	workspace := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_WORKSPACE)
	environment := createEnvironment(t, ctx, agentsClient, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:    orgID,
		Name:              fmt.Sprintf("e2e-workspace-only-%s", uuid.NewString()[:8]),
		RunnerId:          catalogRunnerID(t, ctx),
		WorkspaceImageId:  workspace.GetMeta().GetId(),
		WorkspaceImageTag: discoveredTag(t, ctx, workspace.GetMeta().GetId()),
	})

	_, err := agentsClient.CreateAgent(ctx, &agentsv1.CreateAgentRequest{
		Name:           fmt.Sprintf("e2e-no-runtime-%s", uuid.NewString()[:8]),
		Role:           "assistant",
		Model:          gatewayModelID(t),
		OrganizationId: orgID,
		EnvironmentId:  environment.GetMeta().GetId(),
		Availability:   agentsv1.AgentAvailability_AGENT_AVAILABILITY_INTERNAL,
	})
	if err == nil {
		t.Fatal("CreateAgent succeeded on a workspace-only environment")
	}
	if !strings.Contains(err.Error(), "agent runtime") {
		t.Fatalf("error does not say what is missing: %v", err)
	}
}

// A tag discovery has not seen must be refused on write, or the version list is
// advisory and a workload fails at pull time instead.
func TestAnEnvironmentCannotNameAnUndiscoveredTag(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ctx = asOwner(t, ctx)

	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	orgID := gatewayOrganizationID(t)
	workspace := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_WORKSPACE)

	_, err := agentsClient.CreateEnvironment(ctx, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:    orgID,
		Name:              fmt.Sprintf("e2e-bad-tag-%s", uuid.NewString()[:8]),
		RunnerId:          catalogRunnerID(t, ctx),
		WorkspaceImageId:  workspace.GetMeta().GetId(),
		WorkspaceImageTag: "no-such-tag-ever",
	})
	if err == nil {
		t.Fatal("CreateEnvironment accepted a tag discovery has never seen")
	}
}

// An environment must not name an agent runtime image in the workspace slot:
// the type is what makes the two slots mean anything.
func TestAnEnvironmentRejectsAnImageOfTheWrongType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ctx = asOwner(t, ctx)

	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	orgID := gatewayOrganizationID(t)
	runtime := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_AGENT_RUNTIME)

	_, err := agentsClient.CreateEnvironment(ctx, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:    orgID,
		Name:              fmt.Sprintf("e2e-wrong-type-%s", uuid.NewString()[:8]),
		RunnerId:          catalogRunnerID(t, ctx),
		WorkspaceImageId:  runtime.GetMeta().GetId(),
		WorkspaceImageTag: discoveredTag(t, ctx, runtime.GetMeta().GetId()),
	})
	if err == nil {
		t.Fatal("CreateEnvironment accepted an agent runtime image as the workspace image")
	}
}

func createEnvironment(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, request *agentsv1.CreateEnvironmentRequest) *agentsv1.Environment {
	t.Helper()
	created, err := client.CreateEnvironment(ctx, request)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	environment := created.GetEnvironment()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := client.DeleteEnvironment(asOwner(t, cleanupCtx), &agentsv1.DeleteEnvironmentRequest{Id: environment.GetMeta().GetId()}); err != nil {
			t.Logf("cleanup DeleteEnvironment: %v", err)
		}
	})
	return environment
}

func createSandbox(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, organizationID, environmentID string) *agentsv1.Sandbox {
	t.Helper()
	created, err := client.CreateSandbox(ctx, &agentsv1.CreateSandboxRequest{
		OrganizationId: organizationID,
		EnvironmentId:  environmentID,
	})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	sandbox := created.GetSandbox()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		ownerCtx := asOwner(t, cleanupCtx)
		if _, err := client.DeleteSandbox(ownerCtx, &agentsv1.DeleteSandboxRequest{Id: sandbox.GetMeta().GetId()}); err != nil {
			t.Logf("cleanup DeleteSandbox: %v", err)
			return
		}
		for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); {
			if _, err := client.GetSandbox(ownerCtx, &agentsv1.GetSandboxRequest{
				Ref: &agentsv1.GetSandboxRequest_Id{Id: sandbox.GetMeta().GetId()},
			}); err != nil {
				return
			}
			time.Sleep(2 * time.Second)
		}
		t.Log("cleanup: sandbox still present; the environment delete may fail")
	})
	return sandbox
}

// waitForWorkloadPod finds the pod the Orchestrator created for a sandbox. It
// polls rather than waiting for Running: the assertion is about what the pod was
// assembled with, which is decided before the first pull finishes.
// catalogKubeClientset reads the Kubernetes API from inside the cluster when the
// suite runs there, and from the caller's kubeconfig when it is run against a
// port-forwarded platform from a laptop.
func catalogKubeClientset(t *testing.T) *kubernetes.Clientset {
	t.Helper()
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			t.Skipf("no Kubernetes access: %v", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("kubernetes client: %v", err)
	}
	return clientset
}

// waitForWorkloadPod finds the pod the Orchestrator created for a sandbox. The
// workload id is assigned by the reconciler rather than at creation, so the
// sandbox is re-read until it reports one and the pod named after it exists.
func waitForWorkloadPod(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, sandbox *agentsv1.Sandbox) *corev1.Pod {
	t.Helper()
	clientset := catalogKubeClientset(t)
	namespace := workloadNamespace(t)
	sandboxID := sandbox.GetMeta().GetId()
	deadline := time.Now().Add(3 * time.Minute)
	lastStatus := agentsv1.SandboxStatus_SANDBOX_STATUS_UNSPECIFIED

	for time.Now().Before(deadline) {
		current, err := client.GetSandbox(ctx, &agentsv1.GetSandboxRequest{
			Ref: &agentsv1.GetSandboxRequest_Id{Id: sandboxID},
		})
		if err != nil {
			t.Fatalf("GetSandbox: %v", err)
		}
		lastStatus = current.GetSandbox().GetStatus()
		// A sandbox that gave up will not produce a pod however long we wait.
		if lastStatus == agentsv1.SandboxStatus_SANDBOX_STATUS_FAILED {
			t.Fatalf("sandbox %s failed before a pod was created", sandboxID)
		}

		if workloadID := current.GetSandbox().GetWorkloadId(); workloadID != "" {
			pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Fatalf("list pods: %v", err)
			}
			for i := range pods.Items {
				if strings.Contains(pods.Items[i].Name, workloadID) {
					return &pods.Items[i]
				}
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("context done waiting for the workload pod: %v", ctx.Err())
		case <-time.After(3 * time.Second):
		}
	}
	t.Fatalf("workload pod never appeared; sandbox status was %v", lastStatus)
	return nil
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
