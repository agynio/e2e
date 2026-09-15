//go:build e2e && svc_gateway

package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	sdk "github.com/openziti/sdk-golang"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/stretchr/testify/require"

	zitimgmtv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/ziti_management/v1"
)

const (
	zitiRequestTimeout = 30 * time.Second

	// A Gateway deployed from source re-enrolls and re-binds, which takes
	// noticeably longer than a pod that started from an image.
	zitiBindTimeout      = 90 * time.Second
	zitiBindPollInterval = 2 * time.Second
)

func TestZitiMeEndpointAuthenticated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), zitiRequestTimeout)
	defer cancel()

	conn := dialGRPC(t, zitiManagementAddr(t))
	client := zitimgmtv1.NewZitiManagementServiceClient(conn)

	createResp, err := client.CreateAppIdentity(ctx, &zitimgmtv1.CreateAppIdentityRequest{
		IdentityId: uuid.NewString(),
		Slug:       "e2e-gateway",
	})
	require.NoError(t, err)
	require.NotNil(t, createResp)

	zitiIdentityID := strings.TrimSpace(createResp.GetZitiIdentityId())
	require.NotEmpty(t, zitiIdentityID)
	require.NotEmpty(t, createResp.GetIdentityJson())

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), zitiRequestTimeout)
		defer cleanupCancel()
		_, _ = client.DeleteIdentity(cleanupCtx, &zitimgmtv1.DeleteIdentityRequest{ZitiIdentityId: zitiIdentityID})
	})

	zitiConfig := &ziti.Config{}
	require.NoError(t, json.Unmarshal(createResp.GetIdentityJson(), zitiConfig))

	zitiContext, err := ziti.NewContext(zitiConfig)
	require.NoError(t, err)
	t.Cleanup(func() { zitiContext.Close() })

	// The Gateway binds its Ziti service at startup, and a suite that runs
	// against a Gateway deployed from source can reach this line while that is
	// still in flight: the overlay then reports no terminators, or an identity
	// it has not finished registering. Both are the same "not bound yet", so
	// the dial is retried rather than failed on first contact.
	response, err := dialGatewayMe(t, zitiContext)
	require.NoError(t, err)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status 200, got %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload gatewayMePayload
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	require.NotEmpty(t, strings.TrimSpace(payload.IdentityID))
	require.NotEmpty(t, strings.TrimSpace(payload.IdentityType))
}

// dialGatewayMe dials the Gateway's Ziti service until it answers or the
// deadline passes. Only dial failures are retried — once the overlay routes the
// request, whatever the Gateway says about it is the test's answer.
func dialGatewayMe(t *testing.T, zitiContext ziti.Context) (*http.Response, error) {
	t.Helper()

	client := sdk.NewHttpClient(zitiContext, nil)
	deadline := time.Now().Add(zitiBindTimeout)

	for attempt := 1; ; attempt++ {
		requestCtx, requestCancel := context.WithTimeout(context.Background(), 15*time.Second)
		request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, zitiGatewayBaseURL+"/me", nil)
		if err != nil {
			requestCancel()
			return nil, err
		}

		response, err := client.Do(request)
		if err == nil {
			// The body outlives this function, so the context must too.
			t.Cleanup(requestCancel)
			return response, nil
		}
		requestCancel()

		if !isZitiDialFailure(err) || time.Now().After(deadline) {
			return nil, err
		}
		t.Logf("gateway service not dialable yet (attempt %d): %v", attempt, err)
		time.Sleep(zitiBindPollInterval)
	}
}

// isZitiDialFailure reports whether the overlay refused to route the request at
// all, which is what "the Gateway has not finished binding" looks like from the
// dialing side.
func isZitiDialFailure(err error) bool {
	message := err.Error()
	return strings.Contains(message, "unable to dial service") ||
		strings.Contains(message, "has no terminators") ||
		strings.Contains(message, "identity not found by id")
}
