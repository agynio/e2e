//go:build e2e && svc_gateway

package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynMcp_basic(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	mcpName := testAccMcpName()
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynMcpConfig(t, organizationName, agentName, mcpName, "Terraform acceptance MCP", "mcp start"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_mcp.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_mcp.test", "name", mcpName),
					resource.TestCheckResourceAttr("agyn_mcp.test", "command", "mcp start"),
					resource.TestCheckResourceAttr("agyn_mcp.test", "description", "Terraform acceptance MCP"),
					resource.TestCheckResourceAttrSet("agyn_mcp.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_mcp.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynMcp_update(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	mcpName := testAccMcpName()
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynMcpConfig(t, organizationName, agentName, mcpName, "Terraform acceptance MCP", "mcp start"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_mcp.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_mcp.test", "name", mcpName),
					resource.TestCheckResourceAttr("agyn_mcp.test", "command", "mcp start"),
					resource.TestCheckResourceAttr("agyn_mcp.test", "description", "Terraform acceptance MCP"),
					resource.TestCheckResourceAttrSet("agyn_mcp.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_mcp.test", "id"),
				),
			},
			{
				Config: testAccAgynMcpConfig(t, organizationName, agentName, mcpName, "Terraform acceptance MCP updated", "mcp start --updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_mcp.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_mcp.test", "name", mcpName),
					resource.TestCheckResourceAttr("agyn_mcp.test", "command", "mcp start --updated"),
					resource.TestCheckResourceAttr("agyn_mcp.test", "description", "Terraform acceptance MCP updated"),
					resource.TestCheckResourceAttrSet("agyn_mcp.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_mcp.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynMcp_import(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	mcpName := testAccMcpName()
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynMcpConfig(t, organizationName, agentName, mcpName, "Terraform acceptance MCP", "mcp start"),
			},
			{
				ResourceName:      "agyn_mcp.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynMcpConfig(t *testing.T, organizationName, agentName, name, description, command string) string {
	env := testAccEnv(t)

	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

%s

resource "agyn_mcp" "test" {
	  agent_id    = agyn_agent.test.id
	  name        = %q
	  image       = %q
	  command     = %q
	  description = %q
}
`, testAccProviderConfig(t), organizationName, testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", agentName, "Terraform acceptance agent", "Terraform acceptance role"), name, env.AgentImage, command, description)
}

func testAccMcpName() string {
	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	return fmt.Sprintf("mcp_%s", suffix)
}
