//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	startRetryTestTimeout  = 8 * time.Minute
	failedWorkloadTimeout  = 3 * time.Minute
	fastRetryTimeout       = 40 * time.Second
	fastRetryWindow        = 30 * time.Second
	responseWaitTimeout    = 3 * time.Minute
	runnerCleanupTimeout   = 90 * time.Second
	invalidInitImage       = "INVALID_IMAGE_NAME:latest"
	expectedAgentResponse  = "Hi! How are you?"
)

var configInvalidContainerReasons = map[string]struct{}{
	"InvalidImageName":           {},
	"CreateContainerConfigError": {},
	"CreateContainerError":       {},
}

func TestWorkloadStartRetryPolicyFastRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), startRetryTestTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	runnerConn := dialRunnerGRPC(t, runnerAddr)
	runnersConn := dialGRPC(t, runnersAddr)
	usersConn := dialGRPC(t, usersAddr)
	orgsConn := dialGRPC(t, orgsAddr)

	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)
	llmConn := dialGRPC(t, llmAddr)
	llmClient := llmv1.NewLLMServiceClient(llmConn)
	usersClient := usersv1.NewUsersServiceClient(usersConn)
	orgsClient := organizationsv1.NewOrganizationsServiceClient(orgsConn)
	runnerClient := runnerv1.NewRunnerServiceClient(runnerConn)
	runnersClient := runnersv1.NewRunnersServiceClient(runnersConn)

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	threadsCtx := withIdentity(ctx, identityID)
	token := createAPIToken(t, ctx, usersClient, identityID)
	orgID := createTestOrganization(t, ctx, orgsClient, identityID)

	provider := createLLMProvider(t, ctx, llmClient, testLLMEndpointCodex, orgID)
	providerID := provider.GetMeta().GetId()
	if providerID == "" {
		t.Fatal("create llm provider: missing id")
	}
	model := createModel(t, ctx, llmClient, "e2e-model-"+uuid.NewString(), providerID, "simple-hello", orgID)
	modelID := model.GetMeta().GetId()
	if modelID == "" {
		t.Fatal("create model: missing id")
	}

	agent := createAgent(t, ctx, agentsClient, fmt.Sprintf("e2e-start-retry-%s", uuid.NewString()), modelID, orgID, invalidInitImage)
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

	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadID, identityID, "e2e workload start retry")
	sentMessageTime := messageCreatedAt(t, sentMessage)

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

	failureCtx, failureCancel := context.WithTimeout(ctx, failedWorkloadTimeout)
	defer failureCancel()
	failedWorkloads, err := waitForFailedWorkloads(failureCtx, runnersClient, threadID, agentID, 2)
	if err != nil {
		t.Fatalf("wait for failed workloads: %v", err)
	}
	if len(failedWorkloads) < 2 {
		t.Fatalf("expected at least 2 failed workloads, got %d", len(failedWorkloads))
	}
	failedLatest := failedWorkloads[0]
	failedPrevious := failedWorkloads[1]
	assertFailedWorkload(t, failedLatest, threadID, agentID)
	assertFailedWorkload(t, failedPrevious, threadID, agentID)

	allWorkloads, err := listWorkloadsByThread(ctx, runnersClient, threadID, agentID, nil)
	if err != nil {
		t.Fatalf("list workloads by thread: %v", err)
	}
	if len(allWorkloads) < 2 {
		t.Fatalf("expected at least 2 workloads, got %d", len(allWorkloads))
	}
	if allWorkloads[0].GetStatus() != runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED || allWorkloads[1].GetStatus() != runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED {
		t.Fatalf("expected two consecutive failed workloads, got %s and %s", allWorkloads[0].GetStatus(), allWorkloads[1].GetStatus())
	}
	if workloadID(t, allWorkloads[0]) != workloadID(t, failedLatest) || workloadID(t, allWorkloads[1]) != workloadID(t, failedPrevious) {
		t.Fatalf("expected failures to be the most recent workloads")
	}

	threadResp, err := threadsClient.GetThread(threadsCtx, &threadsv1.GetThreadRequest{ThreadId: threadID})
	if err != nil {
		t.Fatalf("get thread: %v", err)
	}
	if threadResp.GetThread() == nil {
		t.Fatal("get thread: nil response")
	}
	if threadResp.GetThread().GetStatus() != threadsv1.ThreadStatus_THREAD_STATUS_ACTIVE {
		t.Fatalf("expected thread status active, got %s", threadResp.GetThread().GetStatus())
	}

	removedAt := workloadRemovedAt(t, failedLatest)
	updateCtx, updateCancel := context.WithTimeout(ctx, 30*time.Second)
	defer updateCancel()
	validInitImage := codexInitImage
	if _, err := agentsClient.UpdateAgent(updateCtx, &agentsv1.UpdateAgentRequest{Id: agentID, InitImage: &validInitImage}); err != nil {
		t.Fatalf("update agent init image: %v", err)
	}

	fastRetryCtx, fastRetryCancel := context.WithTimeout(ctx, fastRetryTimeout)
	defer fastRetryCancel()
	retryWorkload, err := waitForRetryWorkload(fastRetryCtx, runnersClient, threadID, agentID, removedAt)
	if err != nil {
		t.Fatalf("wait for retry workload: %v", err)
	}
	retryCreatedAt := workloadCreatedAt(t, retryWorkload)
	if retryCreatedAt.Sub(removedAt) >= fastRetryWindow {
		t.Fatalf("expected retry within %s, got %s", fastRetryWindow, retryCreatedAt.Sub(removedAt))
	}

	responseCtx, responseCancel := context.WithTimeout(threadsCtx, responseWaitTimeout)
	defer responseCancel()
	agentBody, err := pollForAgentResponse(t, responseCtx, threadsClient, runnerClient, threadID, agentID, labels, sentMessageTime, expectedAgentResponse)
	if err != nil {
		t.Fatalf("wait for agent response: %v", err)
	}
	if agentBody != expectedAgentResponse {
		t.Fatalf("expected agent response %q, got %q", expectedAgentResponse, agentBody)
	}

	for _, failed := range []*runnersv1.Workload{failedLatest, failedPrevious} {
		instanceID := failed.GetInstanceId()
		if instanceID == "" {
			t.Fatalf("failed workload %s missing instance id", workloadID(t, failed))
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(ctx, runnerCleanupTimeout)
		if err := waitForRunnerWorkloadGone(cleanupCtx, runnerClient, instanceID); err != nil {
			cleanupCancel()
			t.Fatalf("wait for runner workload %s cleanup: %v", instanceID, err)
		}
		cleanupCancel()
	}
}

func waitForFailedWorkloads(
	ctx context.Context,
	client runnersv1.RunnersServiceClient,
	threadID string,
	agentID string,
	count int,
) ([]*runnersv1.Workload, error) {
	if count <= 0 {
		return nil, fmt.Errorf("expected positive count, got %d", count)
	}
	var failed []*runnersv1.Workload
	err := pollUntil(ctx, pollInterval, func(ctx context.Context) error {
		workloads, err := listWorkloadsByThread(ctx, client, threadID, agentID, []runnersv1.WorkloadStatus{runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED})
		if err != nil {
			return err
		}
		if len(workloads) < count {
			return fmt.Errorf("expected %d failed workloads, got %d", count, len(workloads))
		}
		for _, workload := range workloads[:count] {
			if err := validateFailedWorkload(workload, threadID, agentID); err != nil {
				return err
			}
		}
		failed = workloads
		return nil
	})
	if err != nil {
		return nil, err
	}
	return failed, nil
}

func listWorkloadsByThread(
	ctx context.Context,
	client runnersv1.RunnersServiceClient,
	threadID string,
	agentID string,
	statuses []runnersv1.WorkloadStatus,
) ([]*runnersv1.Workload, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread id is empty")
	}
	if agentID == "" {
		return nil, fmt.Errorf("agent id is empty")
	}
	pageToken := ""
	workloads := make([]*runnersv1.Workload, 0, 4)
	for {
		resp, err := client.ListWorkloadsByThread(ctx, &runnersv1.ListWorkloadsByThreadRequest{
			ThreadId:  threadID,
			AgentId:   &agentID,
			Statuses:  statuses,
			PageSize:  20,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list workloads by thread: %w", err)
		}
		workloads = append(workloads, resp.GetWorkloads()...)
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return workloads, nil
}

func waitForRetryWorkload(
	ctx context.Context,
	client runnersv1.RunnersServiceClient,
	threadID string,
	agentID string,
	removedAt time.Time,
) (*runnersv1.Workload, error) {
	if removedAt.IsZero() {
		return nil, fmt.Errorf("removed_at is zero")
	}
	var retry *runnersv1.Workload
	err := pollUntil(ctx, pollInterval, func(ctx context.Context) error {
		workloads, err := listWorkloadsByThread(ctx, client, threadID, agentID, nil)
		if err != nil {
			return err
		}
		for _, workload := range workloads {
			if workload.GetStatus() == runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED {
				continue
			}
			createdAt := workloadCreatedAtErr(workload)
			if createdAt.err != nil {
				return createdAt.err
			}
			if createdAt.time.After(removedAt) {
				retry = workload
				return nil
			}
		}
		return fmt.Errorf("retry workload not found")
	})
	if err != nil {
		return nil, err
	}
	return retry, nil
}

func waitForRunnerWorkloadGone(ctx context.Context, client runnerv1.RunnerServiceClient, workloadID string) error {
	if workloadID == "" {
		return fmt.Errorf("workload id is empty")
	}
	return pollUntil(ctx, pollInterval, func(ctx context.Context) error {
		_, err := client.InspectWorkload(ctx, &runnerv1.InspectWorkloadRequest{WorkloadId: workloadID})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return nil
			}
			return err
		}
		return fmt.Errorf("workload %s still present", workloadID)
	})
}

func assertFailedWorkload(t *testing.T, workload *runnersv1.Workload, threadID, agentID string) {
	t.Helper()
	if err := validateFailedWorkload(workload, threadID, agentID); err != nil {
		t.Fatal(err)
	}
}

func validateFailedWorkload(workload *runnersv1.Workload, threadID, agentID string) error {
	if workload == nil {
		return fmt.Errorf("workload is nil")
	}
	meta := workload.GetMeta()
	if meta == nil {
		return fmt.Errorf("workload metadata missing")
	}
	workloadID := meta.GetId()
	if workloadID == "" {
		return fmt.Errorf("workload id missing")
	}
	if workload.GetStatus() != runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED {
		return fmt.Errorf("workload %s status %s is not failed", workloadID, workload.GetStatus())
	}
	if workload.GetThreadId() != threadID {
		return fmt.Errorf("workload %s thread mismatch", workloadID)
	}
	if workload.GetAgentId() != agentID {
		return fmt.Errorf("workload %s agent mismatch", workloadID)
	}
	if workload.GetFailureReason() != runnersv1.WorkloadFailureReason_WORKLOAD_FAILURE_REASON_CONFIG_INVALID {
		return fmt.Errorf("workload %s failure reason %s is not config invalid", workloadID, workload.GetFailureReason())
	}
	if workload.GetFailureMessage() == "" {
		return fmt.Errorf("workload %s failure message is empty", workloadID)
	}
	if workload.GetRemovedAt() == nil {
		return fmt.Errorf("workload %s removed_at missing", workloadID)
	}
	if workload.GetInstanceId() == "" {
		return fmt.Errorf("workload %s instance id missing", workloadID)
	}
	if err := validateConfigInvalidContainers(workloadID, workload.GetContainers()); err != nil {
		return err
	}
	return nil
}

func validateConfigInvalidContainers(workloadID string, containers []*runnersv1.Container) error {
	if len(containers) == 0 {
		return nil
	}
	for _, container := range containers {
		if container == nil {
			return fmt.Errorf("workload %s has nil container", workloadID)
		}
		if container.GetStatus() != runnersv1.ContainerStatus_CONTAINER_STATUS_WAITING {
			continue
		}
		reason := container.GetReason()
		if reason == "" {
			continue
		}
		if _, ok := configInvalidContainerReasons[reason]; ok {
			return nil
		}
	}
	return fmt.Errorf("workload %s missing waiting container with config invalid reason", workloadID)
}

func workloadID(t *testing.T, workload *runnersv1.Workload) string {
	t.Helper()
	if workload == nil {
		t.Fatal("workload is nil")
	}
	meta := workload.GetMeta()
	if meta == nil {
		t.Fatal("workload metadata missing")
	}
	workloadID := meta.GetId()
	if workloadID == "" {
		t.Fatal("workload id missing")
	}
	return workloadID
}

func workloadCreatedAt(t *testing.T, workload *runnersv1.Workload) time.Time {
	t.Helper()
	created := workloadCreatedAtErr(workload)
	if created.err != nil {
		t.Fatal(created.err)
	}
	return created.time
}

type workloadTimeResult struct {
	time time.Time
	err  error
}

func workloadCreatedAtErr(workload *runnersv1.Workload) workloadTimeResult {
	if workload == nil {
		return workloadTimeResult{err: fmt.Errorf("workload is nil")}
	}
	meta := workload.GetMeta()
	if meta == nil {
		return workloadTimeResult{err: fmt.Errorf("workload metadata missing")}
	}
	workloadID := meta.GetId()
	if workloadID == "" {
		return workloadTimeResult{err: fmt.Errorf("workload id missing")}
	}
	createdAt := meta.GetCreatedAt()
	if createdAt == nil {
		return workloadTimeResult{err: fmt.Errorf("workload %s created_at missing", workloadID)}
	}
	return workloadTimeResult{time: createdAt.AsTime()}
}

func workloadRemovedAt(t *testing.T, workload *runnersv1.Workload) time.Time {
	t.Helper()
	if workload == nil {
		t.Fatal("workload is nil")
	}
	meta := workload.GetMeta()
	if meta == nil {
		t.Fatal("workload metadata missing")
	}
	workloadID := meta.GetId()
	if workloadID == "" {
		t.Fatal("workload id missing")
	}
	removedAt := workload.GetRemovedAt()
	if removedAt == nil {
		t.Fatalf("workload %s removed_at missing", workloadID)
	}
	return removedAt.AsTime()
}
