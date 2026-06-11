//go:build e2e && svc_egress_gateway && !(svc_k8s_runner || smoke)

package tests

import (
	"bufio"
	"context"
	"fmt"
	"net"
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
	assertEgressHTTPStatus(t, ctx, client, baseURL+"/unmatched", http.StatusOK)
	assertWebsocketUpgradeRequired(t, baseURL+"/ws")
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

func assertWebsocketUpgradeRequired(t *testing.T, url string) {
	t.Helper()
	requestURL := strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")
	host, path, ok := strings.Cut(requestURL, "/")
	if !ok {
		path = ""
	}
	conn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		t.Fatalf("dial websocket target: %v", err)
	}
	defer conn.Close()
	_, err = fmt.Fprintf(conn, "GET /%s HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n", path, host)
	if err != nil {
		t.Fatalf("write websocket request: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read websocket response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Fatalf("expected websocket upgrade to return 426, got %d", resp.StatusCode)
	}
}
