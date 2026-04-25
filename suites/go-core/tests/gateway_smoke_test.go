//go:build e2e && smoke

package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatewayMeEndpointUnauthenticated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, gatewayEndpoint(t, "me"), nil)
	require.NoError(t, err)

	response, err := newGatewayClient(t).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.StatusCode)
	}
}

func TestGatewayMeEndpointAuthenticated(t *testing.T) {
	payload := fetchGatewayIdentity(t, gatewayAPIToken(t))
	require.NotEmpty(t, payload.IdentityID)
	require.NotEmpty(t, payload.IdentityType)
}
