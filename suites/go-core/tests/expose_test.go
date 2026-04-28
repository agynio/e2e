//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	zitimgmtv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	sdk "github.com/openziti/sdk-golang"
	"github.com/openziti/sdk-golang/ziti"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	exposeTestTimeout         = 8 * time.Minute
	exposeReachabilityTimeout = 90 * time.Second
	exposeRequestTimeout      = 15 * time.Second
	exposeZitiRequestTimeout  = 30 * time.Second
	exposePort                = 3000
	exposeExpectedResponse    = "Hi! How are you?"
)

type exposeWorkloadFixture struct {
	ctx           context.Context
	podName       string
	containerName string
}

type exposeAddResponse struct {
	ID     string `json:"id"`
	Port   int    `json:"port"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

func TestAgentExposeListExec(t *testing.T) {
	fixture := setupExposeTestWorkload(t)

	execCtx, execCancel := context.WithTimeout(fixture.ctx, 2*time.Minute)
	defer execCancel()

	result := execPodCommand(t, execCtx, workloadNamespace(t), fixture.podName, fixture.containerName, []string{"/agyn-bin/cli/agyn", "expose", "list"})
	if result.exitCode != 0 {
		t.Fatalf("expected expose list exit code 0, got %d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
	}
}

func TestAgentExposeAddReachable(t *testing.T) {
	fixture := setupExposeTestWorkload(t)

	body := fmt.Sprintf("expose-e2e-%s", uuid.NewString())
	serveDir := "/tmp/expose-e2e"
	serveScript := fmt.Sprintf(
		"set -e; mkdir -p %[1]s; printf '%%s' \"$1\" > %[1]s/index.html; "+
			"busybox httpd -f -p %[2]d -h %[1]s >/tmp/expose-httpd.log 2>&1 & "+
			"pid=$!; i=0; while [ \"$i\" -lt 20 ]; do "+
			"if output=$(busybox wget -q -O - http://127.0.0.1:%[2]d/index.html); then "+
			"if [ \"$output\" = \"$1\" ]; then exit 0; fi; fi; "+
			"if ! kill -0 \"$pid\" 2>/dev/null; then echo \"httpd exited\"; "+
			"cat /tmp/expose-httpd.log 2>/dev/null; exit 1; fi; "+
			"i=$((i+1)); sleep 0.5; done; "+
			"echo \"httpd not ready\"; cat /tmp/expose-httpd.log 2>/dev/null; exit 1",
		serveDir,
		exposePort,
	)
	serveCommand := []string{
		"sh",
		"-c",
		serveScript,
		"expose-httpd",
		body,
	}

	serveCtx, serveCancel := context.WithTimeout(fixture.ctx, 2*time.Minute)
	defer serveCancel()
	serveResult := execPodCommand(t, serveCtx, workloadNamespace(t), fixture.podName, fixture.containerName, serveCommand)
	if serveResult.exitCode != 0 {
		t.Fatalf("start http server exit code %d stdout=%q stderr=%q", serveResult.exitCode, serveResult.stdout, serveResult.stderr)
	}

	addCtx, addCancel := context.WithTimeout(fixture.ctx, 2*time.Minute)
	defer addCancel()
	addResult := execPodCommand(t, addCtx, workloadNamespace(t), fixture.podName, fixture.containerName, []string{"/agyn-bin/cli/agyn", "--output", "json", "expose", "add", fmt.Sprintf("%d", exposePort)})
	if addResult.exitCode != 0 {
		t.Fatalf("expose add exit code %d stdout=%q stderr=%q", addResult.exitCode, addResult.stdout, addResult.stderr)
	}

	addResponse := parseExposeAddResponse(t, addResult.stdout)
	if addResponse.ID == "" {
		t.Fatal("expose add missing id")
	}
	if addResponse.Port != exposePort {
		t.Fatalf("expose add port mismatch: got %d want %d", addResponse.Port, exposePort)
	}
	expectedURL := fmt.Sprintf("http://exposed-%s.ziti:%d", addResponse.ID, exposePort)
	if addResponse.URL != expectedURL {
		t.Fatalf("expose add url mismatch: got %q want %q", addResponse.URL, expectedURL)
	}
	if addResponse.Status == "" {
		t.Fatal("expose add status missing")
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cleanupCancel()
		removeResult := execPodCommand(t, cleanupCtx, workloadNamespace(t), fixture.podName, fixture.containerName, []string{"/agyn-bin/cli/agyn", "expose", "remove", fmt.Sprintf("%d", exposePort)})
		if removeResult.exitCode != 0 {
			t.Logf("cleanup: expose remove exit code %d stdout=%q stderr=%q", removeResult.exitCode, removeResult.stdout, removeResult.stderr)
		}
	})

	parsedURL, err := url.Parse(addResponse.URL)
	if err != nil {
		t.Fatalf("parse expose url: %v", err)
	}
	if parsedURL.Scheme != "http" {
		t.Fatalf("expected expose url scheme http, got %q", parsedURL.Scheme)
	}
	if parsedURL.Port() != fmt.Sprintf("%d", exposePort) {
		t.Fatalf("expected expose url port %d, got %q", exposePort, parsedURL.Port())
	}
	host := strings.TrimSpace(parsedURL.Hostname())
	if host == "" {
		t.Fatal("expose url host missing")
	}
	serviceName := strings.TrimSuffix(host, ".ziti")
	if serviceName == host {
		t.Fatalf("expected expose url host to end with .ziti, got %q", host)
	}
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		t.Fatal("expose service name missing")
	}

	exposedURL := fmt.Sprintf("http://%s:%d/index.html", serviceName, exposePort)

	zitiConn := dialGRPC(t, zitiManagementAddr(t))
	zitiClient := zitimgmtv1.NewZitiManagementServiceClient(zitiConn)

	zitiCtx, zitiCancel := context.WithTimeout(context.Background(), exposeZitiRequestTimeout)
	defer zitiCancel()
	createResp, err := zitiClient.CreateAppIdentity(zitiCtx, &zitimgmtv1.CreateAppIdentityRequest{
		IdentityId: uuid.NewString(),
		Slug:       fmt.Sprintf("e2e-expose-%s", uuid.NewString()),
	})
	if err != nil {
		t.Fatalf("create ziti identity: %v", err)
	}
	if createResp == nil {
		t.Fatal("create ziti identity: missing response")
	}
	zitiIdentityID := strings.TrimSpace(createResp.GetZitiIdentityId())
	if zitiIdentityID == "" {
		t.Fatal("create ziti identity: missing id")
	}
	if len(createResp.GetIdentityJson()) == 0 {
		t.Fatal("create ziti identity: missing identity json")
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), exposeZitiRequestTimeout)
		defer cleanupCancel()
		if _, err := zitiClient.DeleteIdentity(cleanupCtx, &zitimgmtv1.DeleteIdentityRequest{ZitiIdentityId: zitiIdentityID}); err != nil {
			t.Logf("cleanup: delete ziti identity %s: %v", zitiIdentityID, err)
		}
	})

	zitiConfig := &ziti.Config{}
	if err := json.Unmarshal(createResp.GetIdentityJson(), zitiConfig); err != nil {
		t.Fatalf("parse ziti identity json: %v", err)
	}

	zitiContext, err := ziti.NewContext(zitiConfig)
	if err != nil {
		t.Fatalf("create ziti context: %v", err)
	}
	t.Cleanup(func() { zitiContext.Close() })

	httpClient := sdk.NewHttpClient(zitiContext, nil)
	httpClient.Timeout = exposeRequestTimeout

	pollCtx, pollCancel := context.WithTimeout(fixture.ctx, exposeReachabilityTimeout)
	defer pollCancel()
	err = pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		requestCtx, requestCancel := context.WithTimeout(ctx, exposeRequestTimeout)
		defer requestCancel()

		request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, exposedURL, nil)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}

		response, err := httpClient.Do(request)
		if err != nil {
			return fmt.Errorf("dial exposed service: %w", err)
		}
		defer response.Body.Close()

		bodyBytes, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status %d (read body error: %v)", response.StatusCode, readErr)
			}
			return fmt.Errorf("read response body: %w", readErr)
		}
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
		}
		responseBody := string(bodyBytes)
		if responseBody != body {
			return fmt.Errorf("unexpected body %q (expected %q)", responseBody, body)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("wait for exposed service: %v", err)
	}
}

func setupExposeTestWorkload(t *testing.T) exposeWorkloadFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), exposeTestTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	runnerConn := dialRunnerGRPC(t, runnerAddr)
	usersConn := dialGRPC(t, usersAddr)
	orgsConn := dialGRPC(t, orgsAddr)

	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)
	llmConn := dialGRPC(t, llmAddr)
	llmClient := llmv1.NewLLMServiceClient(llmConn)
	usersClient := usersv1.NewUsersServiceClient(usersConn)
	orgsClient := organizationsv1.NewOrganizationsServiceClient(orgsConn)
	runnerClient := runnerv1.NewRunnerServiceClient(runnerConn)
	exposeInitImage := envOrDefault("AGN_EXPOSE_INIT_IMAGE", "ghcr.io/agynio/agent-init-agn:latest")

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	threadsCtx := withIdentity(ctx, identityID)
	token := createAPIToken(t, ctx, usersClient, identityID)
	orgID := createTestOrganization(t, ctx, orgsClient, identityID)

	provider := createLLMProvider(t, ctx, llmClient, testLLMEndpointAgn, orgID)
	providerID := provider.GetMeta().GetId()
	if providerID == "" {
		t.Fatal("create llm provider: missing id")
	}
	model := createModel(t, ctx, llmClient, "e2e-expose-model-"+uuid.NewString(), providerID, "simple-hello", orgID)
	modelID := model.GetMeta().GetId()
	if modelID == "" {
		t.Fatal("create model: missing id")
	}

	agent := createAgent(t, ctx, agentsClient, fmt.Sprintf("e2e-expose-%s", uuid.NewString()), modelID, orgID, exposeInitImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() { deleteAgent(t, ctx, agentsClient, agentID) })
	createAgentEnv(t, ctx, agentsClient, agentID, "LLM_API_TOKEN", token)

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentID})
	threadID := thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	labels := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentID,
		labelThreadID:  threadID,
	}
	t.Cleanup(func() {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			t.Logf("cleanup: find workloads: %v", err)
			return
		}
		for _, workloadID := range ids {
			cleanupWorkload(t, ctx, runnerClient, workloadID)
		}
	})

	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadID, identityID, "hi")
	sentMessageTime := messageCreatedAt(t, sentMessage)

	pollCtx, pollCancel := context.WithTimeout(threadsCtx, 5*time.Minute)
	defer pollCancel()
	agentBody, err := pollForAgentResponse(t, pollCtx, threadsClient, runnerClient, threadID, agentID, labels, sentMessageTime, exposeExpectedResponse)
	if err != nil {
		t.Fatalf("wait for agent response: %v", err)
	}
	if agentBody != exposeExpectedResponse {
		t.Fatalf("expected agent response %q, got %q", exposeExpectedResponse, agentBody)
	}

	workloadIDs, err := findWorkloadsByLabels(ctx, runnerClient, labels)
	if err != nil {
		t.Fatalf("find workloads: %v", err)
	}
	if len(workloadIDs) == 0 {
		t.Fatal("expected workload id")
	}

	execCtx, execCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer execCancel()
	podName, containerName, err := waitForWorkloadAgentContainerReady(t, execCtx, workloadIDs[0])
	if err != nil {
		t.Fatalf("wait for agent container: %v", err)
	}

	return exposeWorkloadFixture{
		ctx:           ctx,
		podName:       podName,
		containerName: containerName,
	}
}

func parseExposeAddResponse(t *testing.T, output string) exposeAddResponse {
	t.Helper()
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		t.Fatal("expose add output is empty")
	}
	var response exposeAddResponse
	if err := json.Unmarshal([]byte(trimmed), &response); err != nil {
		t.Fatalf("parse expose add output: %v stdout=%q", err, trimmed)
	}
	response.ID = strings.TrimSpace(response.ID)
	response.URL = strings.TrimSpace(response.URL)
	response.Status = strings.TrimSpace(response.Status)
	return response
}

func waitForWorkloadAgentContainerReady(t *testing.T, ctx context.Context, workloadID string) (string, string, error) {
	t.Helper()
	namespace := workloadNamespace(t)
	podName := fmt.Sprintf("workload-%s", workloadID)
	clientset := kubeClientset(t)
	containerName := ""
	err := pollUntil(ctx, pollInterval, func(ctx context.Context) error {
		pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get pod %s: %w", podName, err)
		}
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("pod %s not running", pod.Name)
		}
		name, err := agentContainerName(pod)
		if err != nil {
			return err
		}
		if !containerReady(pod, name) {
			return fmt.Errorf("container %s not ready", name)
		}
		containerName = name
		return nil
	})
	if err != nil {
		return "", "", err
	}
	return podName, containerName, nil
}
