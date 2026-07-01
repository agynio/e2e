//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	startRetryTestTimeout  = 8 * time.Minute
	failedWorkloadTimeout  = 3 * time.Minute
	fastRetryTimeout       = 40 * time.Second
	fastRetryWindow        = 30 * time.Second
	startRetryPollInterval = time.Second
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
	logStep := startTimingBreadcrumbs(t)
	ctx, cancel := context.WithTimeout(context.Background(), startRetryTestTimeout)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	runnerConn := dialRunnerGRPC(t, runnerAddr)
	runnersConn := dialGRPC(t, runnersAddr)

	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)
	runnerClient := runnerv1.NewRunnerServiceClient(runnerConn)
	runnersClient := runnersv1.NewRunnersServiceClient(runnersConn)

	setup := newWorkflowGatewaySetup(t, ctx)
	identityID := setup.IdentityID
	threadsCtx := setup.Context
	orgID := setup.OrganizationID
	token := setup.Token
	modelID := createWorkflowGatewayModel(t, setup, testLLMEndpointCodex, llmv1.Protocol_PROTOCOL_RESPONSES, "simple-hello")
	logStep("workflow_setup")

	agent := createAgent(t, threadsCtx, agentsClient, fmt.Sprintf("e2e-start-retry-%s", uuid.NewString()), modelID, orgID, invalidInitImage)
	agentID := agent.GetMeta().GetId()
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}
	t.Cleanup(func() {
		cleanupAgentEnvs(t, threadsCtx, agentsClient, agentID)
		deleteAgent(t, threadsCtx, agentsClient, agentID)
	})
	createAgentEnv(t, threadsCtx, agentsClient, agentID, "LLM_API_TOKEN", token)

	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{agentID})
	threadID := thread.GetId()
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadID) })

	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadID, identityID, "hello")
	sentMessageTime := messageCreatedAt(t, sentMessage)
	logStep("invalid_image_message_sent")

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
	logStep("failed_workloads_observed")
	failedLatest := failedWorkloads[0]
	failedPrevious := failedWorkloads[1]
	assertFailedWorkload(t, failedLatest, threadID, agentID)
	assertFailedWorkload(t, failedPrevious, threadID, agentID)

	allWorkloads, err := listWorkloadsByThread(ctx, runnersClient, threadID, agentID, nil)
	if err != nil {
		t.Fatalf("list workloads by thread: %v", err)
	}
	sortedAllWorkloads, err := sortWorkloadsByCreatedAt(allWorkloads, false)
	if err != nil {
		t.Fatalf("sort workloads by created_at: %v", err)
	}
	if len(sortedAllWorkloads) < 2 {
		t.Fatalf("expected at least 2 workloads, got %d", len(allWorkloads))
	}
	if sortedAllWorkloads[0].GetStatus() != runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED || sortedAllWorkloads[1].GetStatus() != runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED {
		t.Fatalf("expected two consecutive failed workloads, got %s and %s", sortedAllWorkloads[0].GetStatus(), sortedAllWorkloads[1].GetStatus())
	}
	if workloadID(t, sortedAllWorkloads[0]) != workloadID(t, failedLatest) || workloadID(t, sortedAllWorkloads[1]) != workloadID(t, failedPrevious) {
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
	logStep("valid_image_updated")

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
	logStep("retry_workload_observed")

	responseCtx, responseCancel := context.WithTimeout(threadsCtx, responseWaitTimeout)
	defer responseCancel()
	agentBody, err := pollForAgentResponse(t, responseCtx, threadsClient, runnerClient, threadID, agentID, labels, sentMessageTime, expectedAgentResponse)
	if err != nil {
		t.Fatalf("wait for agent response: %v", err)
	}
	if agentBody != expectedAgentResponse {
		t.Fatalf("expected agent response %q, got %q", expectedAgentResponse, agentBody)
	}
	logStep("agent_response_observed")

	failedInstanceIDs := make([]string, 0, 2)
	for _, failed := range []*runnersv1.Workload{failedLatest, failedPrevious} {
		instanceID := failed.GetInstanceId()
		if instanceID == "" {
			t.Fatalf("failed workload %s missing instance id", workloadID(t, failed))
		}
		failedInstanceIDs = append(failedInstanceIDs, instanceID)
	}
	cleanupCtx, cleanupCancel := context.WithTimeout(ctx, runnerCleanupTimeout)
	if err := waitForRunnerWorkloadsGone(cleanupCtx, runnerClient, failedInstanceIDs); err != nil {
		cleanupCancel()
		t.Fatalf("wait for runner workload cleanup: %v", err)
	}
	cleanupCancel()
	logStep("failed_runner_workloads_cleaned")
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
	err := pollUntil(ctx, startRetryPollInterval, func(ctx context.Context) error {
		workloads, err := listWorkloadsByThread(ctx, client, threadID, agentID, []runnersv1.WorkloadStatus{runnersv1.WorkloadStatus_WORKLOAD_STATUS_FAILED})
		if err != nil {
			return err
		}
		sortedWorkloads, err := sortWorkloadsByCreatedAt(workloads, false)
		if err != nil {
			return err
		}
		if len(sortedWorkloads) < count {
			return fmt.Errorf("expected %d failed workloads, got %d", count, len(workloads))
		}
		for _, workload := range sortedWorkloads[:count] {
			if err := validateFailedWorkload(workload, threadID, agentID); err != nil {
				return err
			}
		}
		failed = sortedWorkloads
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
	err := pollUntil(ctx, startRetryPollInterval, func(ctx context.Context) error {
		workloads, err := listWorkloadsByThread(ctx, client, threadID, agentID, nil)
		if err != nil {
			return err
		}
		candidate, err := firstWorkloadAfter(workloads, removedAt)
		if err != nil {
			return err
		}
		retry = candidate
		return nil
	})
	if err != nil {
		return nil, err
	}
	return retry, nil
}

type workloadSortEntry struct {
	workload  *runnersv1.Workload
	createdAt time.Time
	id        string
}

func sortWorkloadsByCreatedAt(workloads []*runnersv1.Workload, ascending bool) ([]*runnersv1.Workload, error) {
	entries, err := buildWorkloadSortEntries(workloads)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].createdAt.Equal(entries[j].createdAt) {
			if ascending {
				return entries[i].id < entries[j].id
			}
			return entries[i].id > entries[j].id
		}
		if ascending {
			return entries[i].createdAt.Before(entries[j].createdAt)
		}
		return entries[i].createdAt.After(entries[j].createdAt)
	})
	sorted := make([]*runnersv1.Workload, 0, len(entries))
	for _, entry := range entries {
		sorted = append(sorted, entry.workload)
	}
	return sorted, nil
}

func firstWorkloadAfter(workloads []*runnersv1.Workload, after time.Time) (*runnersv1.Workload, error) {
	if after.IsZero() {
		return nil, fmt.Errorf("after time is zero")
	}
	entries, err := buildWorkloadSortEntries(workloads)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].createdAt.Equal(entries[j].createdAt) {
			return entries[i].id < entries[j].id
		}
		return entries[i].createdAt.Before(entries[j].createdAt)
	})
	for _, entry := range entries {
		if entry.createdAt.After(after) {
			return entry.workload, nil
		}
	}
	return nil, fmt.Errorf("no workload created after %s", after.Format(time.RFC3339Nano))
}

func buildWorkloadSortEntries(workloads []*runnersv1.Workload) ([]workloadSortEntry, error) {
	entries := make([]workloadSortEntry, 0, len(workloads))
	for _, workload := range workloads {
		if workload == nil {
			return nil, fmt.Errorf("workload is nil")
		}
		meta := workload.GetMeta()
		if meta == nil {
			return nil, fmt.Errorf("workload metadata missing")
		}
		workloadID := meta.GetId()
		if workloadID == "" {
			return nil, fmt.Errorf("workload id missing")
		}
		createdAt := meta.GetCreatedAt()
		if createdAt == nil {
			return nil, fmt.Errorf("workload %s created_at missing", workloadID)
		}
		entries = append(entries, workloadSortEntry{
			workload:  workload,
			createdAt: createdAt.AsTime(),
			id:        workloadID,
		})
	}
	return entries, nil
}

func waitForRunnerWorkloadGone(ctx context.Context, client runnerv1.RunnerServiceClient, workloadID string) error {
	if workloadID == "" {
		return fmt.Errorf("workload id is empty")
	}
	return waitForRunnerWorkloadsGone(ctx, client, []string{workloadID})
}

func waitForRunnerWorkloadsGone(ctx context.Context, client runnerv1.RunnerServiceClient, workloadIDs []string) error {
	if len(workloadIDs) == 0 {
		return fmt.Errorf("workload ids are empty")
	}
	remaining := make(map[string]struct{}, len(workloadIDs))
	for _, workloadID := range workloadIDs {
		if workloadID == "" {
			return fmt.Errorf("workload id is empty")
		}
		remaining[workloadID] = struct{}{}
	}
	return pollUntil(ctx, startRetryPollInterval, func(ctx context.Context) error {
		for workloadID := range remaining {
			gone, err := runnerWorkloadGone(ctx, client, workloadID)
			if err != nil {
				return err
			}
			if gone {
				delete(remaining, workloadID)
			}
		}
		if len(remaining) == 0 {
			return nil
		}
		return fmt.Errorf("workloads still present: %s", sortedMapKeys(remaining))
	})
}

func runnerWorkloadGone(ctx context.Context, client runnerv1.RunnerServiceClient, workloadID string) (bool, error) {
	_, err := client.InspectWorkload(ctx, &runnerv1.InspectWorkloadRequest{WorkloadId: workloadID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func sortedMapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
