//go:build e2e && svc_k8s_runner

package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
)

func TestErrors(t *testing.T) {
	client := newK8sRunnerClient(t)

	t.Run("start_workload_missing_image", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		_, err := client.StartWorkload(ctx, &runnerv1.StartWorkloadRequest{
			Main: &runnerv1.ContainerSpec{Image: ""},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("inspect_nonexistent_workload", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		_, err := client.InspectWorkload(ctx, &runnerv1.InspectWorkloadRequest{WorkloadId: "missing-workload"})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("stream_logs_missing_container_name", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		stream, err := client.StreamWorkloadLogs(ctx, &runnerv1.StreamWorkloadLogsRequest{WorkloadId: "missing-workload"})
		require.NoError(t, err)

		_, err = stream.Recv()
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("exec_on_nonexistent_workload", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		stream, err := client.Exec(ctx)
		require.NoError(t, err)

		err = stream.Send(&runnerv1.ExecRequest{Msg: &runnerv1.ExecRequest_Start{Start: &runnerv1.ExecStartRequest{
			TargetId:    "missing-workload",
			CommandArgv: []string{"echo", "hi"},
		}}})
		require.NoError(t, err)

		resp, err := stream.Recv()
		require.NoError(t, err)
		errResp := resp.GetError()
		require.NotNil(t, errResp)
		require.Equal(t, "exec_start_failed", errResp.GetCode())
	})

	t.Run("remove_nonexistent_volume", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		_, err := client.RemoveVolume(ctx, &runnerv1.RemoveVolumeRequest{VolumeName: "missing-volume"})
		requireGRPCCode(t, err, codes.NotFound)
	})
}
