//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// An agent's own image is deprecated and nothing reads it -- an agent runs the
// runtime image its environment names -- but the field is still required on the
// resource, so the suite supplies the runtime the environments here register.
const accAgentImage = "ghcr.io/agynio/agyn-runtime-codex:latest"

type accEnv struct {
	BaseURL    string
	ModelID    string
	AgentImage string
}

func testAccEnv(t *testing.T) accEnv {
	t.Helper()

	return accEnv{
		BaseURL:    requireEnv(t, "AGYN_BASE_URL"),
		ModelID:    requireModelID(t),
		AgentImage: accAgentImage,
	}
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()

	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Fatalf("Missing required environment variable %s", key)
	}

	return value
}

func requireAPIToken(t *testing.T) string {
	return requireEnv(t, "AGYN_API_TOKEN")
}

func requireOrganizationID(t *testing.T) string {
	return requireEnv(t, "AGYN_ORGANIZATION_ID")
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	_ = testAccEnv(t)
}

func testAccOrganizationPreCheck(t *testing.T) {
	t.Helper()

	testAccPreCheck(t)
	requireAPIToken(t)
}

func testAccExternalProviders() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"agyn": {
			Source: "agynio/agyn",
		},
	}
}

func testAccProviderConfig(t *testing.T) string {
	env := testAccEnv(t)

	return fmt.Sprintf(`
provider "agyn" {
  api_url = %q
}
`, env.BaseURL)
}
