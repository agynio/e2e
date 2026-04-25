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
resource "agyn_agent" "test" {
	  organization_id = %s
	  name        = %q
	  description = %q
	  role        = %q
	  model       = %q
	  image       = %q
	  init_image  = %q
}
`, organizationID, name, description, role, env.ModelID, env.AgentImage, env.AgentInitImage)
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
