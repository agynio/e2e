//go:build e2e && (svc_runners || smoke)

package tests

import (
	"context"
	"strings"
	"sync"
	"testing"

	authorizationv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/authorization/v1"
	runnersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runners/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authorizationIdentityPrefix     = "identity:"
	authorizationOrganizationPrefix = "organization:"
	authorizationMemberRelation     = "member"
	authorizationClusterRelation    = "cluster"
	authorizationAdminRelation      = "admin"
	authorizationGlobalCluster      = "cluster:global"

	identityMetadataKey     = "x-identity-id"
	identityTypeMetadataKey = "x-identity-type"
	identityTypeUser        = "user"
	identityTypeAgent       = "agent"

	runnerContainerImage = "alpine:3.21"
)

var (
	authorizationAddr      = envOrDefault("AUTHORIZATION_ADDRESS", "authorization:50051")
	ensureClusterAdminOnce sync.Once
	ensureClusterAdminErr  error
)

func newRunnerClient(t *testing.T) runnersv1.RunnersServiceClient {
	t.Helper()
	conn := dialGRPC(t, runnersAddr)
	return runnersv1.NewRunnersServiceClient(conn)
}

func newAuthorizationClient(t *testing.T) authorizationv1.AuthorizationServiceClient {
	t.Helper()
	conn := dialGRPC(t, authorizationAddr)
	return authorizationv1.NewAuthorizationServiceClient(conn)
}

func contextWithIdentity(ctx context.Context, identityID string, identityType string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		identityMetadataKey, identityID,
		identityTypeMetadataKey, identityType,
	))
}

func adminContext(ctx context.Context) context.Context {
	return contextWithIdentity(ctx, clusterAdminIdentityID, identityTypeUser)
}

func agentContext(ctx context.Context, agentID string) context.Context {
	return contextWithIdentity(ctx, agentID, identityTypeAgent)
}

func ensureClusterAdmin(t *testing.T, ctx context.Context, authzClient authorizationv1.AuthorizationServiceClient) {
	t.Helper()
	ensureClusterAdminOnce.Do(func() {
		tuple := &authorizationv1.TupleKey{
			User:     authorizationIdentityPrefix + clusterAdminIdentityID,
			Relation: authorizationAdminRelation,
			Object:   authorizationGlobalCluster,
		}
		if _, err := authzClient.Write(ctx, &authorizationv1.WriteRequest{Writes: []*authorizationv1.TupleKey{tuple}}); err != nil {
			statusErr, ok := status.FromError(err)
			if !ok {
				ensureClusterAdminErr = err
				return
			}
			if statusErr.Code() == codes.InvalidArgument && strings.Contains(statusErr.Message(), "already exists") {
				return
			}
			ensureClusterAdminErr = err
		}
	})
	if ensureClusterAdminErr != nil {
		t.Fatalf("authorization write failed: %v", ensureClusterAdminErr)
	}
}

func ensureOrganizationMember(t *testing.T, ctx context.Context, authzClient authorizationv1.AuthorizationServiceClient, identityID string, organizationID string) {
	t.Helper()
	ensureClusterAdmin(t, ctx, authzClient)
	memberTuple := &authorizationv1.TupleKey{
		User:     authorizationIdentityPrefix + identityID,
		Relation: authorizationMemberRelation,
		Object:   authorizationOrganizationPrefix + organizationID,
	}
	clusterTuple := &authorizationv1.TupleKey{
		User:     authorizationGlobalCluster,
		Relation: authorizationClusterRelation,
		Object:   authorizationOrganizationPrefix + organizationID,
	}
	tuples := []*authorizationv1.TupleKey{memberTuple, clusterTuple}
	if _, err := authzClient.Write(ctx, &authorizationv1.WriteRequest{Writes: tuples}); err != nil {
		t.Fatalf("authorization write failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), testTimeout)
		defer cleanupCancel()
		cleanupCtx = adminContext(cleanupCtx)
		if _, err := authzClient.Write(cleanupCtx, &authorizationv1.WriteRequest{Deletes: tuples}); err != nil {
			t.Logf("cleanup: authorization delete failed for identity %s org %s: %v", identityID, organizationID, err)
		}
	})
}

func runnerDefaultContainers() []*runnersv1.Container {
	return []*runnersv1.Container{
		{
			ContainerId: "main",
			Name:        "main",
			Role:        runnersv1.ContainerRole_CONTAINER_ROLE_MAIN,
			Image:       runnerContainerImage,
			Status:      runnersv1.ContainerStatus_CONTAINER_STATUS_RUNNING,
		},
	}
}
