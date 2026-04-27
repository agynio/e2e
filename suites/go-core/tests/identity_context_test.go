//go:build e2e

package tests

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func withIdentity(ctx context.Context, identityID string) context.Context {
	md := metadata.New(map[string]string{"x-identity-id": identityID})
	return metadata.NewOutgoingContext(ctx, md)
}
