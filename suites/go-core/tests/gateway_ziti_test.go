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

const zitiRequestTimeout = 30 * time.Second

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

	requestCtx, requestCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer requestCancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, zitiGatewayBaseURL+"/me", nil)
	require.NoError(t, err)

	response, err := sdk.NewHttpClient(zitiContext, nil).Do(request)
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
