//go:build e2e && svc_k8s_runner

package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
)

func TestVolumeQueries(t *testing.T) {
	client := newK8sRunnerClient(t)

	t.Run("list_workloads_by_volume", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		volumeName := uniqueName("volume")
		workloadID := startWorkload(t, ctx, client, volumeWorkloadRequest(volumeName))
		waitRunning(t, ctx, client, workloadID)

		resp, err := client.ListWorkloadsByVolume(ctx, &runnerv1.ListWorkloadsByVolumeRequest{VolumeName: volumeName})
		require.NoError(t, err)
		require.Contains(t, resp.GetTargetIds(), workloadID)
	})

	t.Run("remove_volume", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		volumeName := uniqueName("volume")
		workloadID := startWorkload(t, ctx, client, volumeWorkloadRequest(volumeName))
		waitRunning(t, ctx, client, workloadID)

		_, err := client.StopWorkload(ctx, &runnerv1.StopWorkloadRequest{WorkloadId: workloadID, TimeoutSec: 1})
		require.NoError(t, err)
		waitGone(t, ctx, client, workloadID)

		_, err = client.RemoveVolume(ctx, &runnerv1.RemoveVolumeRequest{VolumeName: volumeName})
		require.NoError(t, err)
		_, err = client.RemoveVolume(ctx, &runnerv1.RemoveVolumeRequest{VolumeName: volumeName})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func volumeWorkloadRequest(volumeName string) *runnerv1.StartWorkloadRequest {
	req := sleepWorkloadRequest()
	req.Volumes = []*runnerv1.VolumeSpec{{
		Name:           "data",
		Kind:           runnerv1.VolumeKind_VOLUME_KIND_NAMED,
		PersistentName: volumeName,
	}}
	req.Main.Mounts = []*runnerv1.VolumeMount{{
		Volume:    "data",
		MountPath: "/data",
	}}
	return req
}
