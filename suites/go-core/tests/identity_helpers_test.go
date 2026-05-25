//go:build e2e

package tests

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func withIdentity(ctx context.Context, identityID string) context.Context {
	return contextWithIdentity(ctx, identityID, identityTypeUser)
}

func withAgentIdentity(ctx context.Context, agentID string) context.Context {
	return contextWithIdentity(ctx, agentID, identityTypeAgent)
}

func contextWithIdentity(ctx context.Context, identityID string, identityType string) context.Context {
	md := metadata.New(map[string]string{
		identityMetadataKey:     identityID,
		identityTypeMetadataKey: identityType,
	})
	return metadata.NewOutgoingContext(ctx, md)
}
