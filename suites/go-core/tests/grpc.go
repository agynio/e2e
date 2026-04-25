//go:build e2e && (svc_agents_orchestrator || svc_runners || svc_metering || svc_k8s_runner || svc_organizations || svc_files || svc_media_proxy || smoke)

package tests

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// dialGRPC creates an insecure gRPC connection. The test fails immediately on error.
func dialGRPC(t *testing.T, addr string, opts ...grpc.DialOption) *grpc.ClientConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	options = append(options, opts...)
	conn, err := grpc.DialContext(ctx, addr, options...)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}
