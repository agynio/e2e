//go:build e2e && smoke

package tests

import (
	"context"
	"testing"
	"time"

	filesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/files/v1"
	"google.golang.org/grpc/codes"
)

func TestFilesSmokeMetadataRequiresID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := newFilesClient(t)
	_, err := client.GetFileMetadata(ctx, &filesv1.GetFileMetadataRequest{})
	requireFilesGRPCCode(t, err, codes.InvalidArgument)
}
