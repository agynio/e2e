//go:build e2e && svc_runners

package tests

import (
	"context"
	"testing"
	"time"

	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBatchUpdateWorkloadSampledAtSingle(t *testing.T) {
	runnerClient := newRunnerClient(t)
	authzClient := newAuthorizationClient(t)
	ctx, runnerID := setupRunnerFixture(t, runnerClient)

	threadID := uuid.NewString()
	agentID := uuid.NewString()
	organizationID := uuid.NewString()
	ensureOrganizationMember(t, ctx, authzClient, clusterAdminIdentityID, organizationID)

	workloadID := uuid.NewString()
	createRunnerWorkload(t, ctx, runnerClient, workloadID, runnerID, threadID, agentID, organizationID)
	cleanupRunnerWorkload(t, runnerClient, workloadID)

	sampledAt := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := runnerClient.BatchUpdateWorkloadSampledAt(ctx, &runnersv1.BatchUpdateWorkloadSampledAtRequest{
		Entries: []*runnersv1.SampledAtEntry{{
			Id:        workloadID,
			SampledAt: timestamppb.New(sampledAt),
		}},
	}); err != nil {
		t.Fatalf("BatchUpdateWorkloadSampledAt failed: %v", err)
	}

	assertRunnerWorkloadSampledAt(t, ctx, runnerClient, workloadID, sampledAt)
}

func TestBatchUpdateWorkloadSampledAtMultiple(t *testing.T) {
	runnerClient := newRunnerClient(t)
	authzClient := newAuthorizationClient(t)
	ctx, runnerID := setupRunnerFixture(t, runnerClient)

	threadID := uuid.NewString()
	agentID := uuid.NewString()
	organizationID := uuid.NewString()
	ensureOrganizationMember(t, ctx, authzClient, clusterAdminIdentityID, organizationID)

	workloadIDs := []string{uuid.NewString(), uuid.NewString()}
	for _, workloadID := range workloadIDs {
		createRunnerWorkload(t, ctx, runnerClient, workloadID, runnerID, threadID, agentID, organizationID)
		cleanupRunnerWorkload(t, runnerClient, workloadID)
	}

	baseTime := time.Now().UTC().Truncate(time.Microsecond)
	workloadTimes := []time.Time{baseTime, baseTime.Add(2 * time.Minute)}

	entries := make([]*runnersv1.SampledAtEntry, 0, len(workloadIDs))
	for i, workloadID := range workloadIDs {
		entries = append(entries, &runnersv1.SampledAtEntry{
			Id:        workloadID,
			SampledAt: timestamppb.New(workloadTimes[i]),
		})
	}

	if _, err := runnerClient.BatchUpdateWorkloadSampledAt(ctx, &runnersv1.BatchUpdateWorkloadSampledAtRequest{
		Entries: entries,
	}); err != nil {
		t.Fatalf("BatchUpdateWorkloadSampledAt failed: %v", err)
	}

	for i, workloadID := range workloadIDs {
		assertRunnerWorkloadSampledAt(t, ctx, runnerClient, workloadID, workloadTimes[i])
	}
}

func TestBatchUpdateWorkloadSampledAtIdempotent(t *testing.T) {
	runnerClient := newRunnerClient(t)
	authzClient := newAuthorizationClient(t)
	ctx, runnerID := setupRunnerFixture(t, runnerClient)

	threadID := uuid.NewString()
	agentID := uuid.NewString()
	organizationID := uuid.NewString()
	ensureOrganizationMember(t, ctx, authzClient, clusterAdminIdentityID, organizationID)

	workloadID := uuid.NewString()
	createRunnerWorkload(t, ctx, runnerClient, workloadID, runnerID, threadID, agentID, organizationID)
	cleanupRunnerWorkload(t, runnerClient, workloadID)

	sampledAt := time.Now().UTC().Truncate(time.Microsecond)
	req := &runnersv1.BatchUpdateWorkloadSampledAtRequest{
		Entries: []*runnersv1.SampledAtEntry{{
			Id:        workloadID,
			SampledAt: timestamppb.New(sampledAt),
		}},
	}

	if _, err := runnerClient.BatchUpdateWorkloadSampledAt(ctx, req); err != nil {
		t.Fatalf("BatchUpdateWorkloadSampledAt failed: %v", err)
	}
	if _, err := runnerClient.BatchUpdateWorkloadSampledAt(ctx, req); err != nil {
		t.Fatalf("BatchUpdateWorkloadSampledAt idempotent failed: %v", err)
	}

	assertRunnerWorkloadSampledAt(t, ctx, runnerClient, workloadID, sampledAt)
}

func TestBatchUpdateVolumeSampledAtSingle(t *testing.T) {
	runnerClient := newRunnerClient(t)
	authzClient := newAuthorizationClient(t)
	ctx, runnerID := setupRunnerFixture(t, runnerClient)

	threadID := uuid.NewString()
	agentID := uuid.NewString()
	organizationID := uuid.NewString()
	ensureOrganizationMember(t, ctx, authzClient, clusterAdminIdentityID, organizationID)

	volumeID := uuid.NewString()
	volumeExternalID := uuid.NewString()
	createRunnerVolume(t, ctx, runnerClient, volumeID, volumeExternalID, runnerID, threadID, agentID, organizationID)
	cleanupRunnerVolume(t, runnerClient, volumeID)

	sampledAt := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := runnerClient.BatchUpdateVolumeSampledAt(ctx, &runnersv1.BatchUpdateVolumeSampledAtRequest{
		Entries: []*runnersv1.SampledAtEntry{{
			Id:        volumeID,
			SampledAt: timestamppb.New(sampledAt),
		}},
	}); err != nil {
		t.Fatalf("BatchUpdateVolumeSampledAt failed: %v", err)
	}

	assertRunnerVolumeSampledAt(t, ctx, runnerClient, volumeID, sampledAt)
}

func TestBatchUpdateVolumeSampledAtMultiple(t *testing.T) {
	runnerClient := newRunnerClient(t)
	authzClient := newAuthorizationClient(t)
	ctx, runnerID := setupRunnerFixture(t, runnerClient)

	threadID := uuid.NewString()
	agentID := uuid.NewString()
	organizationID := uuid.NewString()
	ensureOrganizationMember(t, ctx, authzClient, clusterAdminIdentityID, organizationID)

	volumeIDs := []string{uuid.NewString(), uuid.NewString()}
	volumeExternalIDs := []string{uuid.NewString(), uuid.NewString()}
	for i, volumeID := range volumeIDs {
		createRunnerVolume(t, ctx, runnerClient, volumeID, volumeExternalIDs[i], runnerID, threadID, agentID, organizationID)
		cleanupRunnerVolume(t, runnerClient, volumeID)
	}

	baseTime := time.Now().UTC().Truncate(time.Microsecond)
	volumeTimes := []time.Time{baseTime.Add(3 * time.Minute), baseTime.Add(5 * time.Minute)}

	entries := make([]*runnersv1.SampledAtEntry, 0, len(volumeIDs))
	for i, volumeID := range volumeIDs {
		entries = append(entries, &runnersv1.SampledAtEntry{
			Id:        volumeID,
			SampledAt: timestamppb.New(volumeTimes[i]),
		})
	}

	if _, err := runnerClient.BatchUpdateVolumeSampledAt(ctx, &runnersv1.BatchUpdateVolumeSampledAtRequest{
		Entries: entries,
	}); err != nil {
		t.Fatalf("BatchUpdateVolumeSampledAt failed: %v", err)
	}

	for i, volumeID := range volumeIDs {
		assertRunnerVolumeSampledAt(t, ctx, runnerClient, volumeID, volumeTimes[i])
	}
}

func TestBatchUpdateVolumeSampledAtIdempotent(t *testing.T) {
	runnerClient := newRunnerClient(t)
	authzClient := newAuthorizationClient(t)
	ctx, runnerID := setupRunnerFixture(t, runnerClient)

	threadID := uuid.NewString()
	agentID := uuid.NewString()
	organizationID := uuid.NewString()
	ensureOrganizationMember(t, ctx, authzClient, clusterAdminIdentityID, organizationID)

	volumeID := uuid.NewString()
	volumeExternalID := uuid.NewString()
	createRunnerVolume(t, ctx, runnerClient, volumeID, volumeExternalID, runnerID, threadID, agentID, organizationID)
	cleanupRunnerVolume(t, runnerClient, volumeID)

	sampledAt := time.Now().UTC().Truncate(time.Microsecond)
	req := &runnersv1.BatchUpdateVolumeSampledAtRequest{
		Entries: []*runnersv1.SampledAtEntry{{
			Id:        volumeID,
			SampledAt: timestamppb.New(sampledAt),
		}},
	}

	if _, err := runnerClient.BatchUpdateVolumeSampledAt(ctx, req); err != nil {
		t.Fatalf("BatchUpdateVolumeSampledAt failed: %v", err)
	}
	if _, err := runnerClient.BatchUpdateVolumeSampledAt(ctx, req); err != nil {
		t.Fatalf("BatchUpdateVolumeSampledAt idempotent failed: %v", err)
	}

	assertRunnerVolumeSampledAt(t, ctx, runnerClient, volumeID, sampledAt)
}

func setupRunnerFixture(t *testing.T, runnerClient runnersv1.RunnersServiceClient) (context.Context, string) {
	t.Helper()
	ctx := newRunnerTestContext(t)
	runnerID := registerRunnerFixture(t, ctx, runnerClient)
	cleanupRunnerFixture(t, runnerClient, runnerID)
	return ctx, runnerID
}

func newRunnerTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return adminContext(ctx)
}

func cleanupRunnerFixture(t *testing.T, runnerClient runnersv1.RunnersServiceClient, runnerID string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), testTimeout)
		defer cleanupCancel()
		cleanupCtx = adminContext(cleanupCtx)
		_, _ = runnerClient.DeleteRunner(cleanupCtx, &runnersv1.DeleteRunnerRequest{Id: runnerID})
	})
}

func cleanupRunnerWorkload(t *testing.T, runnerClient runnersv1.RunnersServiceClient, workloadID string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), testTimeout)
		defer cleanupCancel()
		cleanupCtx = adminContext(cleanupCtx)
		_, _ = runnerClient.DeleteWorkload(cleanupCtx, &runnersv1.DeleteWorkloadRequest{Id: workloadID})
	})
}

func cleanupRunnerVolume(t *testing.T, runnerClient runnersv1.RunnersServiceClient, volumeID string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), testTimeout)
		defer cleanupCancel()
		cleanupCtx = adminContext(cleanupCtx)
		_, _ = runnerClient.UpdateVolume(cleanupCtx, &runnersv1.UpdateVolumeRequest{
			Id:        volumeID,
			Status:    runnersv1.VolumeStatus_VOLUME_STATUS_DELETED.Enum(),
			RemovedAt: timestamppb.New(time.Now().UTC()),
		})
	})
}

func assertRunnerWorkloadSampledAt(t *testing.T, ctx context.Context, runnerClient runnersv1.RunnersServiceClient, workloadID string, expected time.Time) {
	t.Helper()
	resp, err := runnerClient.GetWorkload(ctx, &runnersv1.GetWorkloadRequest{Id: workloadID})
	if err != nil {
		t.Fatalf("GetWorkload failed: %v", err)
	}
	workload := resp.GetWorkload()
	if workload == nil {
		t.Fatal("GetWorkload missing workload")
	}
	lastSampledAt := workload.GetLastMeteringSampledAt()
	if lastSampledAt == nil {
		t.Fatalf("workload %s missing last_metering_sampled_at (expected %s)", workloadID, expected)
	}
	actual := lastSampledAt.AsTime()
	if !actual.Equal(expected) {
		t.Fatalf("workload %s sampled_at mismatch: got %s, want %s", workloadID, actual, expected)
	}
}

func assertRunnerVolumeSampledAt(t *testing.T, ctx context.Context, runnerClient runnersv1.RunnersServiceClient, volumeID string, expected time.Time) {
	t.Helper()
	resp, err := runnerClient.GetVolume(ctx, &runnersv1.GetVolumeRequest{Id: volumeID})
	if err != nil {
		t.Fatalf("GetVolume failed: %v", err)
	}
	volume := resp.GetVolume()
	if volume == nil {
		t.Fatal("GetVolume missing volume")
	}
	lastSampledAt := volume.GetLastMeteringSampledAt()
	if lastSampledAt == nil {
		t.Fatalf("volume %s missing last_metering_sampled_at (expected %s)", volumeID, expected)
	}
	actual := lastSampledAt.AsTime()
	if !actual.Equal(expected) {
		t.Fatalf("volume %s sampled_at mismatch: got %s, want %s", volumeID, actual, expected)
	}
}

func registerRunnerFixture(t *testing.T, ctx context.Context, runnerClient runnersv1.RunnersServiceClient) string {
	t.Helper()
	resp, err := runnerClient.RegisterRunner(ctx, &runnersv1.RegisterRunnerRequest{
		Name: "e2e-runner-" + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("RegisterRunner failed: %v", err)
	}
	runner := resp.GetRunner()
	if runner == nil || runner.GetMeta() == nil {
		t.Fatal("runner metadata missing")
	}
	runnerID := runner.GetMeta().GetId()
	if runnerID == "" {
		t.Fatal("runner ID missing")
	}
	return runnerID
}

func createRunnerWorkload(t *testing.T, ctx context.Context, runnerClient runnersv1.RunnersServiceClient, workloadID, runnerID, threadID, agentID, organizationID string) {
	t.Helper()
	resp, err := runnerClient.CreateWorkload(ctx, &runnersv1.CreateWorkloadRequest{
		Id:             workloadID,
		RunnerId:       runnerID,
		ThreadId:       threadID,
		AgentId:        agentID,
		OrganizationId: organizationID,
		Status:         runnersv1.WorkloadStatus_WORKLOAD_STATUS_RUNNING,
		Containers:     runnerDefaultContainers(),
		ZitiIdentityId: "ziti-test",
	})
	if err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}
	workload := resp.GetWorkload()
	if workload == nil || workload.GetMeta() == nil {
		t.Fatal("CreateWorkload missing workload metadata")
	}
	if workload.GetMeta().GetId() != workloadID {
		t.Fatalf("CreateWorkload returned unexpected ID")
	}
}

func createRunnerVolume(t *testing.T, ctx context.Context, runnerClient runnersv1.RunnersServiceClient, volumeID, volumeExternalID, runnerID, threadID, agentID, organizationID string) {
	t.Helper()
	resp, err := runnerClient.CreateVolume(ctx, &runnersv1.CreateVolumeRequest{
		Id:             volumeID,
		VolumeId:       volumeExternalID,
		ThreadId:       threadID,
		RunnerId:       runnerID,
		AgentId:        agentID,
		OrganizationId: organizationID,
		SizeGb:         "10",
		Status:         runnersv1.VolumeStatus_VOLUME_STATUS_PROVISIONING,
	})
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}
	volume := resp.GetVolume()
	if volume == nil || volume.GetMeta() == nil {
		t.Fatal("CreateVolume missing volume metadata")
	}
	if volume.GetMeta().GetId() != volumeID {
		t.Fatalf("CreateVolume returned unexpected ID")
	}
}
