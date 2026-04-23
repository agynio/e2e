//go:build e2e && svc_k8s_runner

package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
)

func TestStreaming(t *testing.T) {
	client := newK8sRunnerClient(t)

	t.Run("logs_follow", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		req := &runnerv1.StartWorkloadRequest{
			Main: &runnerv1.ContainerSpec{
				Image:      defaultWorkloadImage,
				Entrypoint: "/bin/sh",
				Cmd:        []string{"-c", "echo follow-1; echo follow-2; sleep 2"},
			},
		}
		workloadID := startWorkload(t, ctx, client, req)
		waitRunning(t, ctx, client, workloadID)

		logs := collectWorkloadLogs(t, ctx, client, workloadID, true, 0)
		require.Contains(t, logs, "follow-1")
		require.Contains(t, logs, "follow-2")
	})

	t.Run("logs_tail", func(t *testing.T) {
		ctx, cancel := testContext(t)
		t.Cleanup(cancel)

		req := &runnerv1.StartWorkloadRequest{
			Main: &runnerv1.ContainerSpec{
				Image:      defaultWorkloadImage,
				Entrypoint: "/bin/sh",
				Cmd:        []string{"-c", "echo line-1; echo line-2; echo line-3; echo line-4; sleep 2"},
			},
		}
		workloadID := startWorkload(t, ctx, client, req)
		waitRunning(t, ctx, client, workloadID)

		logs := collectWorkloadLogs(t, ctx, client, workloadID, false, 2)
		require.Contains(t, logs, "line-3")
		require.Contains(t, logs, "line-4")
		require.NotContains(t, logs, "line-1")
	})
}

func TestStreamEvents(t *testing.T) {
	ctx, cancel := testContext(t)
	t.Cleanup(cancel)

	client := newK8sRunnerClient(t)
	streamCtx, streamCancel := context.WithTimeout(ctx, waitRunningTimeout)
	t.Cleanup(streamCancel)

	stream, err := client.StreamEvents(streamCtx, &runnerv1.StreamEventsRequest{})
	require.NoError(t, err)

	workloadID := startWorkload(t, ctx, client, sleepWorkloadRequest())
	waitRunning(t, ctx, client, workloadID)
	podName := podNameFromID(workloadID)

	for {
		resp, err := stream.Recv()
		require.NoError(t, err)
		if data := resp.GetData(); data != nil {
			if strings.Contains(data.GetJson(), podName) {
				return
			}
			continue
		}
		if errResp := resp.GetError(); errResp != nil {
			t.Fatalf("events stream error: %s", errResp.GetMessage())
		}
	}
}
