//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"strings"
	"testing"
)

func testAccAgynAgentResourceBlock(t *testing.T, organizationID, name, description, role string) string {
	env := testAccEnv(t)

	return fmt.Sprintf(`
resource "agyn_runner" "test" {
	  organization_id = %s
	  name            = "tf-acc-runner"
	  capabilities    = ["docker"]
}

resource "agyn_image" "test" {
	  organization_id = %s
	  name            = "tf-acc-workspace"
	  type            = "workspace"
	  repository      = "ghcr.io/agynio/devcontainer"
	  visibility      = "internal"
}

resource "agyn_image" "runtime" {
	  organization_id = %s
	  name            = "tf-acc-runtime"
	  type            = "agent_runtime"
	  repository      = "ghcr.io/agynio/agyn-runtime-codex"
	  visibility      = "internal"
}

resource "agyn_environment" "test" {
	  organization_id     = %s
	  name                = "tf-acc-environment"
	  runner_id           = agyn_runner.test.id
	  workspace_image_id      = agyn_image.test.id
	  workspace_image_tag     = "latest"
	  agent_runtime_image_id  = agyn_image.runtime.id
	  agent_runtime_image_tag = "latest"
}

resource "agyn_agent" "test" {
	  organization_id = %s
	  environment_id  = agyn_environment.test.id
	  name         = %q
	  description  = %q
	  role         = %q
	  model        = %q
	  image        = %q
	  availability = "internal"
}
`, organizationID, organizationID, organizationID, organizationID, organizationID, name, description, role, env.ModelID, env.AgentImage)
}

func formatCapabilitiesLine(capabilities []string, indent string) string {
	if len(capabilities) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		quoted = append(quoted, fmt.Sprintf("%q", capability))
	}
	return fmt.Sprintf("\n%s%s", indent, fmt.Sprintf("capabilities = [%s]", strings.Join(quoted, ", ")))
}
