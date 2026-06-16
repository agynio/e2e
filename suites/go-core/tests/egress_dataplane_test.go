//go:build e2e && svc_egress_gateway && !(svc_k8s_runner || smoke)

package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	assertEgressInjectedHeader(t, ctx, client, baseURL+"/allowed", "X-E2E-Egress-Injected")
	assertEgressInjectedHeader(t, ctx, client, baseURL+"/literal-header", "X-E2E-Egress-Literal")
	assertEgressInjectedHeader(t, ctx, client, baseURL+"/secret-header", "X-E2E-Egress-Secret")
	assertEgressHTTPStatus(t, ctx, client, baseURL+"/denied", http.StatusForbidden)
	assertUnmatchedBypassesGateway(t, ctx, client, baseURL)
	assertWebsocketUpgradeRequired(t, ctx, client, baseURL+"/ws")
}

func TestEgressGatewayDataPlaneMatcherMatrix(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("EGRESS_DATAPLANE_BASE_URL"), "/")
	if baseURL == "" {
		t.Skip("EGRESS_DATAPLANE_BASE_URL is required for live egress data-plane checks")
	}

	ctx, cancel := context.WithTimeout(context.Background(), egressDataPlaneTimeout)
	t.Cleanup(cancel)

	client := &http.Client{Timeout: 10 * time.Second}
	assertEgressRequest(t, ctx, client, http.MethodGet, baseURL+"/repos/agynio/e2e", http.StatusOK, "X-E2E-Egress-Path-Matched")
	assertEgressRequest(t, ctx, client, http.MethodPost, baseURL+"/repos/agynio/e2e", http.StatusOK, "X-E2E-Egress-Method-Matched")
	assertEgressRequest(t, ctx, client, http.MethodGet, baseURL+"/ports/8443", http.StatusOK, "X-E2E-Egress-Port-Matched")
	assertEgressRequest(t, ctx, client, http.MethodGet, baseURL+"/repos", http.StatusNotFound, "")
}

func assertEgressHTTPStatus(t *testing.T, ctx context.Context, client *http.Client, url string, expected int) {
	t.Helper()
	assertEgressRequest(t, ctx, client, http.MethodGet, url, expected, "")
}

func assertEgressInjectedHeader(t *testing.T, ctx context.Context, client *http.Client, url string, markerHeader string) {
	t.Helper()
	assertEgressRequest(t, ctx, client, http.MethodGet, url, http.StatusOK, markerHeader)
}

func assertEgressRequest(t *testing.T, ctx context.Context, client *http.Client, method string, url string, expectedStatus int, expectedHeader string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		t.Fatalf("build egress request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform egress request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected %s %s to return %d, got %d", method, url, expectedStatus, resp.StatusCode)
	}
	if expectedHeader != "" && resp.Header.Get(expectedHeader) == "" {
		t.Fatalf("expected %s %s response to expose %s marker", method, url, expectedHeader)
	}
}

func assertUnmatchedBypassesGateway(t *testing.T, ctx context.Context, client *http.Client, dataPlaneBaseURL string) {
	t.Helper()
	directURL := strings.TrimSpace(os.Getenv("EGRESS_DIRECT_BYPASS_URL"))
	if directURL == "" {
		t.Fatal("EGRESS_DIRECT_BYPASS_URL is required to prove unmatched destinations bypass the gateway")
	}
	assertDistinctURLOrigin(t, dataPlaneBaseURL, directURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, directURL, nil)
	if err != nil {
		t.Fatalf("build direct bypass request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform direct bypass request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected direct bypass request to return %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if resp.Header.Get("X-E2E-Egress-Injected") != "" {
		t.Fatalf("direct bypass request included gateway injected header marker")
	}
	if resp.Header.Get("X-E2E-Egress-Gateway") != "" {
		t.Fatalf("direct bypass request included gateway marker")
	}
	assertExpectedDirectBypassHeader(t, resp.Header)
}

func assertDistinctURLOrigin(t *testing.T, dataPlaneBaseURL string, directURL string) {
	t.Helper()
	dataPlaneOrigin, err := urlOrigin(dataPlaneBaseURL)
	if err != nil {
		t.Fatalf("parse EGRESS_DATAPLANE_BASE_URL: %v", err)
	}
	directOrigin, err := urlOrigin(directURL)
	if err != nil {
		t.Fatalf("parse EGRESS_DIRECT_BYPASS_URL: %v", err)
	}
	if dataPlaneOrigin == directOrigin {
		t.Fatalf("EGRESS_DIRECT_BYPASS_URL origin %s must differ from EGRESS_DATAPLANE_BASE_URL origin", directOrigin)
	}
}

func urlOrigin(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("absolute URL required")
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), nil
}

func assertExpectedDirectBypassHeader(t *testing.T, header http.Header) {
	t.Helper()
	headerName := strings.TrimSpace(os.Getenv("EGRESS_DIRECT_BYPASS_HEADER"))
	if headerName == "" {
		return
	}
	expectedValue := strings.TrimSpace(os.Getenv("EGRESS_DIRECT_BYPASS_HEADER_VALUE"))
	actualValue := header.Get(headerName)
	if actualValue == "" {
		t.Fatalf("expected direct bypass response header %s", headerName)
	}
	if expectedValue != "" && actualValue != expectedValue {
		t.Fatalf("expected direct bypass response header %s=%q, got %q", headerName, expectedValue, actualValue)
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
