//go:build e2e && (svc_agents_orchestrator || smoke)

package tests

import (
	"testing"

	"google.golang.org/grpc"
)

func dialRunnerGRPC(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	return dialGRPC(t, addr)
}
