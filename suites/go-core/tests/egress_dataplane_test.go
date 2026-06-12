//go:build e2e && svc_egress_gateway && !(svc_k8s_runner || smoke)

package tests

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const egressDataPlaneTimeout = 2 * time.Minute

func TestEgressGatewayDataPlaneHTTPBehavior(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("EGRESS_DATAPLANE_BASE_URL"), "/")
	if baseURL == "" {
		t.Skip("EGRESS_DATAPLANE_BASE_URL is required for live egress data-plane checks")
	}

	ctx, cancel := context.WithTimeout(context.Background(), egressDataPlaneTimeout)
	t.Cleanup(cancel)

	client := &http.Client{Timeout: 10 * time.Second}
	assertEgressHTTPStatus(t, ctx, client, baseURL+"/allowed", http.StatusOK)
	assertEgressInjectedHeader(t, ctx, client, baseURL+"/allowed")
	assertEgressHTTPStatus(t, ctx, client, baseURL+"/denied", http.StatusForbidden)
	assertUnmatchedBypassesGateway(t, ctx, client, baseURL+"/unmatched")
	assertWebsocketUpgradeRequired(t, ctx, client, baseURL+"/ws")
}

func assertEgressHTTPStatus(t *testing.T, ctx context.Context, client *http.Client, url string, expected int) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform egress request %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expected {
		t.Fatalf("expected %s to return %d, got %d", url, expected, resp.StatusCode)
	}
}

func assertEgressInjectedHeader(t *testing.T, ctx context.Context, client *http.Client, url string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build injected-header request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform injected-header request: %v", err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-E2E-Egress-Injected") == "" {
		t.Fatalf("expected response to expose X-E2E-Egress-Injected marker")
	}
}

func assertUnmatchedBypassesGateway(t *testing.T, ctx context.Context, client *http.Client, url string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build unmatched request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform unmatched request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected unmatched request to return %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if resp.Header.Get("X-E2E-Egress-Injected") != "" {
		t.Fatalf("unmatched request included gateway injected header marker")
	}
	if resp.Header.Get("X-E2E-Egress-Gateway") != "" {
		t.Fatalf("unmatched request included gateway marker")
	}
}

func assertWebsocketUpgradeRequired(t *testing.T, ctx context.Context, client *http.Client, url string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build websocket request: %v", err)
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform websocket request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Fatalf("expected websocket upgrade to return 426, got %d", resp.StatusCode)
	}
}
