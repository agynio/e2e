//go:build e2e && svc_egress && !(svc_runners || smoke)

package tests

import (
	"context"
	"sync"
	"testing"

	authorizationv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/authorization/v1"
	"google.golang.org/grpc/status"
)

const (
	authorizationIdentityPrefix     = "identity:"
	authorizationOrganizationPrefix = "organization:"
	authorizationMemberRelation     = "member"
	authorizationAdminRelation      = "admin"
	authorizationGlobalCluster      = "cluster:global"
)

var (
	authorizationAddr      = envOrDefault("AUTHORIZATION_ADDRESS", "authorization:50051")
	ensureClusterAdminOnce sync.Once
	ensureClusterAdminErr  error
)

func newAuthorizationClient(t *testing.T) authorizationv1.AuthorizationServiceClient {
	t.Helper()
	conn := dialGRPC(t, authorizationAddr)
	return authorizationv1.NewAuthorizationServiceClient(conn)
}

func adminContext(ctx context.Context) context.Context {
	return contextWithIdentity(ctx, clusterAdminIdentityID, identityTypeUser)
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
			// AlreadyExists as well as InvalidArgument: authorization answers a
			// duplicate tuple with the former now, and this went on treating it as
			// a failure.
			if isAlreadyWritten(statusErr) {
				return
			}
			ensureClusterAdminErr = err
		}
	})
	if ensureClusterAdminErr != nil {
		t.Fatalf("authorization write failed: %v", ensureClusterAdminErr)
	}
}
