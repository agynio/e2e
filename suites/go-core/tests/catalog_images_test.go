//go:build e2e && (svc_agents_orchestrator || svc_images || smoke)

package tests

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	imagesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/images/v1"
	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// Covers the path an image reference takes end to end: registered in the
// catalog, named by an environment, rewritten by the Orchestrator onto the
// image proxy, and pulled by the runner with a credential minted for that one
// workload. Every step of it used to be a free-form string typed by an operator.

var imagesAddr = envOrDefault("IMAGES_ADDRESS", "images:50051")

// The label the assembler stamps on every sandbox workload.
const assemblerSandboxIDLabel = "sandbox-id"

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
	workspaceTag := discoveredTag(t, ctx, workspace.GetMeta().GetId())

	environment := createEnvironment(t, ctx, agentsClient, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:    orgID,
		Name:              fmt.Sprintf("e2e-catalog-%s", uuid.NewString()[:8]),
		RunnerId:          catalogRunnerID(t, ctx),
		WorkspaceImageId:  workspace.GetMeta().GetId(),
		WorkspaceImageTag: workspaceTag,
	})

	sandbox := createSandbox(t, ctx, agentsClient, orgID, environment.GetMeta().GetId())
	pod := waitForWorkloadPod(t, ctx, sandbox)

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

// The acceptance signal of Environment and Runtime Unification: agynd and the
// agyn CLI arrive from chart-pinned init images, and the agent CLI from the
// environment — so what a workload runs is the platform's build plus one agent
// CLI the environment names, not whatever a single image happened to bundle.
func TestAWorkloadGetsThreeInitContainers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = asOwner(t, ctx)

	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	orgID := gatewayOrganizationID(t)

	workspace := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_WORKSPACE)
	runtime := registerCatalogImageFrom(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_AGENT_RUNTIME,
		envOrDefault("TEST_RUNTIME_REPOSITORY", "ghcr.io/agynio/agyn-runtime-codex"))

	environment := createEnvironment(t, ctx, agentsClient, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:       orgID,
		Name:                 fmt.Sprintf("e2e-three-init-%s", uuid.NewString()[:8]),
		RunnerId:             catalogRunnerID(t, ctx),
		WorkspaceImageId:     workspace.GetMeta().GetId(),
		WorkspaceImageTag:    discoveredTag(t, ctx, workspace.GetMeta().GetId()),
		AgentRuntimeImageId:  runtime.GetMeta().GetId(),
		AgentRuntimeImageTag: discoveredTag(t, ctx, runtime.GetMeta().GetId()),
	})

	sandbox := createSandbox(t, ctx, agentsClient, orgID, environment.GetMeta().GetId())
	pod := waitForWorkloadPod(t, ctx, sandbox)

	present := map[string]string{}
	for _, container := range pod.Spec.InitContainers {
		present[container.Name] = container.Image
	}
	for _, name := range []string{"agynd-cli-init", "agyn-cli-init", "agent-runtime"} {
		if _, ok := present[name]; !ok {
			t.Fatalf("init container %q missing; pod has %v", name, present)
		}
	}
	// The platform's own binaries are pinned by the chart, so they must not come
	// from the proxy or from the agent runtime image.
	for _, name := range []string{"agynd-cli-init", "agyn-cli-init"} {
		if strings.Contains(present[name], envOrDefault("IMAGE_PROXY_HOST", "registry.agyn.dev")) {
			t.Fatalf("%s is pulled through the proxy (%s); it is a platform component", name, present[name])
		}
	}
	if !strings.Contains(present["agent-runtime"], envOrDefault("IMAGE_PROXY_HOST", "registry.agyn.dev")) {
		t.Fatalf("agent-runtime does not pull through the proxy: %s", present["agent-runtime"])
	}
	// The pre-split single init container must be gone.
	if _, ok := present["agent-init"]; ok {
		t.Fatal("the legacy agent-init container is still assembled")
	}

	// What the init containers were for: the platform's binaries and one agent
	// CLI, all in the shared volume the main container sees.
	binaries := sharedBinaries(t, ctx, pod)
	for _, want := range []string{"/agynd", "cli/agyn", "/codex", "/config.json"} {
		if !binaries[want] {
			t.Fatalf("%s missing from the shared volume; it holds %v", want, binaries)
		}
	}
}

// sharedBinaries lists what the init containers left in /agyn/bin, waiting for
// the workload to run first: the volume is only populated once they complete.
func sharedBinaries(t *testing.T, ctx context.Context, pod *corev1.Pod) map[string]bool {
	t.Helper()
	namespace := workloadNamespace(t)
	clientset := catalogKubeClientset(t)

	for deadline := time.Now().Add(4 * time.Minute); time.Now().Before(deadline); {
		current, err := clientset.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get pod: %v", err)
		}
		if current.Status.Phase == corev1.PodRunning {
			// The three images write disjoint paths, so the listing is one level
			// deep: agynd and the agent CLI at the root, the agyn CLI under cli/.
			stdout, err := catalogExec(t, ctx, namespace, pod.Name, current.Spec.Containers[0].Name,
				[]string{"sh", "-c", "cd /agyn/bin && ls . cli 2>/dev/null | sed 's|^|/|'; ls cli 2>/dev/null | sed 's|^|cli/|'"})
			if err != nil {
				t.Fatalf("ls /agyn/bin: %v", err)
			}
			present := map[string]bool{}
			for _, entry := range strings.Fields(stdout) {
				present[entry] = true
			}
			return present
		}
		if current.Status.Phase == corev1.PodFailed {
			t.Fatalf("workload pod failed before running")
		}
		time.Sleep(5 * time.Second)
	}
	t.Fatal("workload pod never reached Running")
	return nil
}

// A workspace-only environment is the sandbox case: the platform binaries
// arrive, and no agent CLI does.
func TestAWorkspaceOnlyWorkloadGetsOnlyThePlatformInitContainers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = asOwner(t, ctx)

	agentsClient := agentsv1.NewAgentsServiceClient(dialGRPC(t, agentsAddr))
	orgID := gatewayOrganizationID(t)
	workspace := registerCatalogImage(t, ctx, orgID, imagesv1.ImageType_IMAGE_TYPE_WORKSPACE)

	environment := createEnvironment(t, ctx, agentsClient, &agentsv1.CreateEnvironmentRequest{
		OrganizationId:    orgID,
		Name:              fmt.Sprintf("e2e-two-init-%s", uuid.NewString()[:8]),
		RunnerId:          catalogRunnerID(t, ctx),
		WorkspaceImageId:  workspace.GetMeta().GetId(),
		WorkspaceImageTag: discoveredTag(t, ctx, workspace.GetMeta().GetId()),
	})

	sandbox := createSandbox(t, ctx, agentsClient, orgID, environment.GetMeta().GetId())
	pod := waitForWorkloadPod(t, ctx, sandbox)

	present := map[string]bool{}
	for _, container := range pod.Spec.InitContainers {
		present[container.Name] = true
	}
	for _, name := range []string{"agynd-cli-init", "agyn-cli-init"} {
		if !present[name] {
			t.Fatalf("init container %q missing from a workspace-only workload", name)
		}
	}
	if present["agent-runtime"] {
		t.Fatal("a workspace-only environment supplied an agent runtime container")
	}

	// The sandbox case: the platform's own CLI is there, and no agent CLI is.
	binaries := sharedBinaries(t, ctx, pod)
	for _, want := range []string{"/agynd", "cli/agyn"} {
		if !binaries[want] {
			t.Fatalf("%s missing from a workspace-only workload; volume holds %v", want, binaries)
		}
	}
	for _, cli := range []string{"/codex", "/claude", "/agn"} {
		if binaries[cli] {
			t.Fatalf("%s is present without the environment naming an agent runtime", cli)
		}
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
		if _, err := client.DeleteSandbox(asOwner(t, cleanupCtx), &agentsv1.DeleteSandboxRequest{Id: sandbox.GetMeta().GetId()}); err != nil {
			t.Logf("cleanup DeleteSandbox: %v", err)
		}
	})
	return sandbox
}

// catalogKubeClientset reads the Kubernetes API from inside the cluster when the
// suite runs there, and from the caller's kubeconfig when it is run against a
// port-forwarded platform from a laptop.
func catalogRestConfig(t *testing.T) *rest.Config {
	t.Helper()
	config, err := rest.InClusterConfig()
	if err == nil {
		return config
	}
	config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		t.Skipf("no Kubernetes access: %v", err)
	}
	return config
}

func catalogKubeClientset(t *testing.T) *kubernetes.Clientset {
	t.Helper()
	clientset, err := kubernetes.NewForConfig(catalogRestConfig(t))
	if err != nil {
		t.Fatalf("kubernetes client: %v", err)
	}
	return clientset
}

// catalogExec runs a command in a workload container. The suite's own helper is
// in-cluster only; this one works from a laptop against a port-forwarded
// platform too, which is how these are run locally.
func catalogExec(t *testing.T, ctx context.Context, namespace, podName, containerName string, command []string) (string, error) {
	t.Helper()
	clientset := catalogKubeClientset(t)
	request := clientset.CoreV1().RESTClient().Post().
		Namespace(namespace).Resource("pods").Name(podName).SubResource("exec")
	request.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(catalogRestConfig(t), "POST", request.URL())
	if err != nil {
		t.Fatalf("create exec: %v", err)
	}
	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		return stdout.String(), fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

// waitForWorkloadPod finds the pod the Orchestrator assembled for a sandbox. It
// matches on the sandbox label rather than waiting for the sandbox to report a
// workload id: that is only written once the pod is Running, and what is under
// test here is how the pod was assembled, which is settled before the first
// image is pulled.
func waitForWorkloadPod(t *testing.T, ctx context.Context, sandbox *agentsv1.Sandbox) *corev1.Pod {
	t.Helper()
	clientset := catalogKubeClientset(t)
	namespace := workloadNamespace(t)
	selector := fmt.Sprintf("%s=%s", assemblerSandboxIDLabel, sandbox.GetMeta().GetId())
	deadline := time.Now().Add(2 * time.Minute)

	for time.Now().Before(deadline) {
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			t.Fatalf("list pods: %v", err)
		}
		if len(pods.Items) > 0 {
			return &pods.Items[0]
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context done waiting for the workload pod: %v", ctx.Err())
		case <-time.After(3 * time.Second):
		}
	}
	t.Fatalf("no workload pod carrying %s", selector)
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
