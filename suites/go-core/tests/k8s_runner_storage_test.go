//go:build e2e && svc_k8s_runner

package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
)

func TestPutArchive(t *testing.T) {
	ctx, cancel := testContext(t)
	t.Cleanup(cancel)

	client := newK8sRunnerClient(t)
	resp := startWorkloadWithCleanup(t, ctx, client, &runnerv1.StartWorkloadRequest{
		Main: &runnerv1.ContainerSpec{
			Image:      defaultWorkloadImage,
			Entrypoint: "/bin/sh",
			Cmd:        []string{"-c", "mkdir -p /tmp/e2e && sleep 300"},
		},
	})
	workloadID := resp.GetId()
	targetID := resp.GetContainers().GetMain()
	require.NotEmpty(t, targetID)
	waitRunning(t, ctx, client, workloadID)

	tarPayload := buildTarWithFile("hello.txt", "hello-storage")
	_, err := client.PutArchive(ctx, &runnerv1.PutArchiveRequest{
		WorkloadId: workloadID,
		Path:       "/tmp/e2e",
		TarPayload: tarPayload,
	})
	require.NoError(t, err)

	result := collectExecOutput(t, ctx, client, &runnerv1.ExecStartRequest{
		TargetId:    targetID,
		CommandArgv: []string{"cat", "/tmp/e2e/hello.txt"},
	})
	require.NotNil(t, result.exit)
	require.Equal(t, int32(0), result.exit.GetExitCode())
	require.Contains(t, result.stdout, "hello-storage")
}
