//go:build e2e && svc_media_proxy

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	filesv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/files/v1"
	"google.golang.org/grpc/metadata"
)

const (
	mediaProxyIdentityMetadataKey     = "x-identity-id"
	mediaProxyIdentityTypeMetadataKey = "x-identity-type"
)

var (
	mediaProxyURL = envOrDefault("MEDIA_PROXY_URL", "http://media-proxy:8080")
	gatewayURL    = envOrDefault("GATEWAY_URL", "http://gateway:8080")

	// Minted from the bundled Keycloak over plain in-cluster HTTP. The token
	// still verifies against the Media Proxy, which is configured with the
	// browser-facing issuer: Keycloak stamps `iss` from KC_HOSTNAME rather than
	// from the address it was called on. Going through the ingress instead would
	// mean trusting the VM's local CA here for no gain.
	oidcTokenURL = envOrDefault("OIDC_TOKEN_URL",
		"http://keycloak.keycloak.svc.cluster.local:8080/realms/agyn/protocol/openid-connect/token")
	oidcClientID = envOrDefault("OIDC_CLIENT_ID", "agyn-chat")
	// The user the realm ships with. It is the cluster admin, which these tests
	// do not rely on — they need an identity that resolves, not a particular
	// permission level.
	oidcUsername = envOrDefault("OIDC_USERNAME", "admin")
	oidcPassword = envOrDefault("OIDC_PASSWORD", "admin")

	accessToken      string
	resolvedIdentity mediaProxyIdentity
)

type mediaProxyIdentityType string

const (
	mediaProxyIdentityTypeUser   mediaProxyIdentityType = "user"
	mediaProxyIdentityTypeAgent  mediaProxyIdentityType = "agent"
	mediaProxyIdentityTypeApp    mediaProxyIdentityType = "app"
	mediaProxyIdentityTypeRunner mediaProxyIdentityType = "runner"
)

type mediaProxyIdentity struct {
	IdentityID   string
	IdentityType mediaProxyIdentityType
}

type mePayload struct {
	IdentityID   string `json:"identity_id"`
	IdentityType string `json:"identity_type"`
}

func TestMain(m *testing.M) {
	cleanup := &cleanupStack{}
	ctx := context.Background()

	if err := setupCredentials(ctx, cleanup); err != nil {
		exitWithSetupError(cleanup, fmt.Errorf("setup credentials: %w", err))
	}

	exitCode := m.Run()
	cleanup.Run()
	os.Exit(exitCode)
}

type cleanupStack struct {
	fns []func()
}

func (c *cleanupStack) Add(fn func()) {
	c.fns = append(c.fns, fn)
}

func (c *cleanupStack) Run() {
	for i := len(c.fns) - 1; i >= 0; i-- {
		c.fns[i]()
	}
}

func exitWithSetupError(cleanup *cleanupStack, err error) {
	cleanup.Run()
	fmt.Fprintf(os.Stderr, "e2e setup failed: %v\n", err)
	os.Exit(1)
}

func setupCredentials(ctx context.Context, _ *cleanupStack) error {
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	token, err := requestOIDCAccessToken(requestCtx)
	if err != nil {
		return err
	}

	meCtx, meCancel := context.WithTimeout(ctx, 15*time.Second)
	defer meCancel()

	identityInfo, err := fetchIdentity(meCtx, token)
	if err != nil {
		return err
	}

	accessToken = token
	resolvedIdentity = identityInfo
	return nil
}

func requestOIDCAccessToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", oidcUsername)
	form.Set("password", oidcPassword)
	form.Set("scope", "openid profile email")
	form.Set("client_id", oidcClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oidcTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := newClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oidc token request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	token := strings.TrimSpace(payload.AccessToken)
	if token == "" {
		return "", fmt.Errorf("oidc access_token missing")
	}

	return token, nil
}

func fetchIdentity(ctx context.Context, token string) (mediaProxyIdentity, error) {
	client := newAuthenticatedClient(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gatewayURL+"/me", nil)
	if err != nil {
		return mediaProxyIdentity{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return mediaProxyIdentity{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return mediaProxyIdentity{}, fmt.Errorf("me endpoint failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload mePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return mediaProxyIdentity{}, err
	}

	identityID := strings.TrimSpace(payload.IdentityID)
	if identityID == "" {
		return mediaProxyIdentity{}, fmt.Errorf("identity_id missing")
	}
	identityType, err := parseIdentityType(payload.IdentityType)
	if err != nil {
		return mediaProxyIdentity{}, err
	}

	return mediaProxyIdentity{IdentityID: identityID, IdentityType: identityType}, nil
}

func parseIdentityType(value string) (mediaProxyIdentityType, error) {
	trimmed := strings.TrimSpace(value)
	switch trimmed {
	case string(mediaProxyIdentityTypeUser):
		return mediaProxyIdentityTypeUser, nil
	case string(mediaProxyIdentityTypeAgent):
		return mediaProxyIdentityTypeAgent, nil
	case string(mediaProxyIdentityTypeApp):
		return mediaProxyIdentityTypeApp, nil
	case string(mediaProxyIdentityTypeRunner):
		return mediaProxyIdentityTypeRunner, nil
	default:
		return "", fmt.Errorf("unsupported identity type: %q", value)
	}
}

func proxyURL(rawURL string, size int) string {
	params := url.Values{}
	params.Set("url", strings.TrimSpace(rawURL))
	if size > 0 {
		params.Set("size", strconv.Itoa(size))
	}

	return mediaProxyURL + "/proxy?" + params.Encode()
}

func uploadTestFile(t *testing.T, ctx context.Context, filename, contentType string, data []byte) string {
	t.Helper()

	resolved := resolvedIdentity
	if resolved.IdentityID == "" {
		t.Fatal("identity id missing")
	}
	if resolved.IdentityType == "" {
		t.Fatal("identity type missing")
	}
	if len(data) == 0 {
		t.Fatal("file data is empty")
	}

	uploadCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		mediaProxyIdentityMetadataKey, resolved.IdentityID,
		mediaProxyIdentityTypeMetadataKey, string(resolved.IdentityType),
	))
	client := newFilesClient(t)
	stream, err := client.UploadFile(uploadCtx)
	if err != nil {
		t.Fatalf("start upload: %v", err)
	}

	metadata := &filesv1.UploadFileRequest{
		Payload: &filesv1.UploadFileRequest_Metadata{
			Metadata: &filesv1.UploadFileMetadata{
				Filename:    filename,
				ContentType: contentType,
				SizeBytes:   int64(len(data)),
			},
		},
	}
	if err := stream.Send(metadata); err != nil {
		t.Fatalf("send metadata: %v", err)
	}

	chunkSplit := len(data) / 2
	if chunkSplit == 0 {
		chunkSplit = len(data)
	}
	if err := stream.Send(&filesv1.UploadFileRequest{
		Payload: &filesv1.UploadFileRequest_Chunk{
			Chunk: &filesv1.UploadFileChunk{Data: data[:chunkSplit]},
		},
	}); err != nil {
		t.Fatalf("send chunk: %v", err)
	}
	if chunkSplit < len(data) {
		if err := stream.Send(&filesv1.UploadFileRequest{
			Payload: &filesv1.UploadFileRequest_Chunk{
				Chunk: &filesv1.UploadFileChunk{Data: data[chunkSplit:]},
			},
		}); err != nil {
			t.Fatalf("send chunk: %v", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("finish upload: %v", err)
	}
	if resp == nil || resp.GetFile() == nil {
		t.Fatal("missing file in upload response")
	}

	fileID := strings.TrimSpace(resp.GetFile().GetId())
	if fileID == "" {
		t.Fatal("missing file id in upload response")
	}

	return fileID
}

func decodeImage(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode image: %v", err)
	}

	return img
}

func newClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func newAuthenticatedClient(token string) *http.Client {
	client := newClient()
	client.Transport = bearerTransport{token: token, base: client.Transport}
	return client
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return base.RoundTrip(clone)
}
