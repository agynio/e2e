//go:build e2e && (svc_runners || smoke)

package tests

import (
	"context"
	"reflect"
	"testing"

	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	"github.com/google/uuid"
)

func TestRunnerLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	adminCtx := adminContext(ctx)

	runnerClient := newRunnerClient(t)
	authzClient := newAuthorizationClient(t)

	registerLabels := map[string]string{
		"region": "test",
		"tier":   "e2e",
	}
	registerResp, err := runnerClient.RegisterRunner(adminCtx, &runnersv1.RegisterRunnerRequest{
		Name:   "e2e-runner-" + uuid.NewString(),
		Labels: registerLabels,
	})
	if err != nil {
		t.Fatalf("RegisterRunner failed: %v", err)
	}

	runner := registerResp.GetRunner()
	if runner == nil || runner.GetMeta() == nil {
		t.Fatal("runner metadata missing")
	}
	runnerID := runner.GetMeta().GetId()
	if runnerID == "" {
		t.Fatal("runner ID missing")
	}
	if !reflect.DeepEqual(runner.GetLabels(), registerLabels) {
		t.Fatalf("RegisterRunner returned unexpected labels")
	}
	token := registerResp.GetServiceToken()
	if token == "" {
		t.Fatal("service token missing")
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), testTimeout)
		defer cleanupCancel()
		cleanupCtx = adminContext(cleanupCtx)
		_, _ = runnerClient.DeleteRunner(cleanupCtx, &runnersv1.DeleteRunnerRequest{Id: runnerID})
	})

	validateResp, err := runnerClient.ValidateServiceToken(adminCtx, &runnersv1.ValidateServiceTokenRequest{
		TokenHash: token,
	})
	if err != nil {
		t.Fatalf("ValidateServiceToken failed: %v", err)
	}
	if validateResp.GetRunner().GetMeta().GetId() != runnerID {
		t.Fatalf("ValidateServiceToken returned unexpected runner ID")
	}

	workloadID := uuid.NewString()
	threadID := uuid.NewString()
	agentID := uuid.NewString()
	organizationID := uuid.NewString()
	ensureOrganizationMember(t, adminCtx, authzClient, clusterAdminIdentityID, organizationID)

	createResp, err := runnerClient.CreateWorkload(adminCtx, &runnersv1.CreateWorkloadRequest{
		Id:             workloadID,
		RunnerId:       runnerID,
		ThreadId:       threadID,
		AgentId:        agentID,
		OrganizationId: organizationID,
		Status:         runnersv1.WorkloadStatus_WORKLOAD_STATUS_STARTING,
		Containers:     runnerDefaultContainers(),
		ZitiIdentityId: "ziti-test",
	})
	if err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), testTimeout)
		defer cleanupCancel()
		cleanupCtx = adminContext(cleanupCtx)
		_, _ = runnerClient.DeleteWorkload(cleanupCtx, &runnersv1.DeleteWorkloadRequest{Id: workloadID})
	})

	if createResp.GetWorkload().GetMeta().GetId() != workloadID {
		t.Fatalf("CreateWorkload returned unexpected ID")
	}

	updateResp, err := runnerClient.UpdateWorkloadStatus(adminCtx, &runnersv1.UpdateWorkloadStatusRequest{
		Id:         workloadID,
		Status:     runnersv1.WorkloadStatus_WORKLOAD_STATUS_RUNNING,
		Containers: runnerDefaultContainers(),
	})
	if err != nil {
		t.Fatalf("UpdateWorkloadStatus failed: %v", err)
	}
	if updateResp.GetWorkload().GetStatus() != runnersv1.WorkloadStatus_WORKLOAD_STATUS_RUNNING {
		t.Fatalf("UpdateWorkloadStatus did not return running status")
	}

	agentCtx := agentContext(ctx, agentID)
	if _, err := runnerClient.TouchWorkload(agentCtx, &runnersv1.TouchWorkloadRequest{Id: workloadID}); err != nil {
		t.Fatalf("TouchWorkload failed: %v", err)
	}

	getResp, err := runnerClient.GetWorkload(adminCtx, &runnersv1.GetWorkloadRequest{Id: workloadID})
	if err != nil {
		t.Fatalf("GetWorkload failed: %v", err)
	}
	if getResp.GetWorkload().GetThreadId() != threadID {
		t.Fatalf("GetWorkload returned unexpected thread ID")
	}

	listByThreadResp, err := runnerClient.ListWorkloadsByThread(adminCtx, &runnersv1.ListWorkloadsByThreadRequest{
		ThreadId:  threadID,
		PageSize:  10,
		PageToken: "",
	})
	if err != nil {
		t.Fatalf("ListWorkloadsByThread failed: %v", err)
	}
	if !containsRunnerWorkload(listByThreadResp.GetWorkloads(), workloadID) {
		t.Fatalf("ListWorkloadsByThread missing workload")
	}

	listResp, err := runnerClient.ListWorkloads(adminCtx, &runnersv1.ListWorkloadsRequest{
		PageSize: 10,
		Statuses: []runnersv1.WorkloadStatus{runnersv1.WorkloadStatus_WORKLOAD_STATUS_RUNNING},
	})
	if err != nil {
		t.Fatalf("ListWorkloads failed: %v", err)
	}
	if !containsRunnerWorkload(listResp.GetWorkloads(), workloadID) {
		t.Fatalf("ListWorkloads missing workload")
	}

	if _, err := runnerClient.DeleteWorkload(adminCtx, &runnersv1.DeleteWorkloadRequest{Id: workloadID}); err != nil {
		t.Fatalf("DeleteWorkload failed: %v", err)
	}

	deletedResp, err := runnerClient.GetWorkload(adminCtx, &runnersv1.GetWorkloadRequest{Id: workloadID})
	if err != nil {
		t.Fatalf("GetWorkload after delete failed: %v", err)
	}
	if deletedResp.GetWorkload().GetStatus() != runnersv1.WorkloadStatus_WORKLOAD_STATUS_STOPPED {
		t.Fatalf("expected stopped status after delete")
	}
	if deletedResp.GetWorkload().GetRemovedAt() == nil {
		t.Fatalf("expected removed_at after delete")
	}

	if _, err := runnerClient.DeleteRunner(adminCtx, &runnersv1.DeleteRunnerRequest{Id: runnerID}); err != nil {
		t.Fatalf("DeleteRunner failed: %v", err)
	}
}

func containsRunnerWorkload(workloads []*runnersv1.Workload, workloadID string) bool {
	for _, workload := range workloads {
		if workload.GetMeta().GetId() == workloadID {
			return true
		}
	}
	return false
}
