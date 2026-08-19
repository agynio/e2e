//go:build e2e

package tests

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

// isAlreadyWritten reports a tuple that is already in the store. Authorization
// answered a duplicate with InvalidArgument and now answers AlreadyExists; the
// helpers that tolerate one were failing on the other.
func isAlreadyWritten(statusErr *status.Status) bool {
	if statusErr == nil {
		return false
	}
	if statusErr.Code() == codes.AlreadyExists {
		return true
	}
	return statusErr.Code() == codes.InvalidArgument && strings.Contains(statusErr.Message(), "already exists")
}
