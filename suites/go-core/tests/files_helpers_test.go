//go:build e2e && (svc_files || svc_media_proxy || smoke)

package tests

import (
	"os"
	"strings"
	"testing"

	filesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/files/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var filesAddr = filesEnvOrDefault("FILES_ADDRESS", "files:50051")

func filesEnvOrDefault(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func newFilesClient(t *testing.T) filesv1.FilesServiceClient {
	t.Helper()
	conn := dialGRPC(t, filesAddr)
	return filesv1.NewFilesServiceClient(conn)
}

func requireFilesGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected gRPC error %s, got nil", code)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != code {
		t.Fatalf("expected %s, got %s: %s", code, st.Code(), st.Message())
	}
}
