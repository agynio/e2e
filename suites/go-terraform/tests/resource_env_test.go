//go:build e2e && svc_gateway

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynEnv_basic(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynEnvConfig(t, organizationName, agentName, "Terraform acceptance env", "env-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_env.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_env.test", "name", "AGYN_ENV"),
					resource.TestCheckResourceAttr("agyn_env.test", "description", "Terraform acceptance env"),
					resource.TestCheckResourceAttrSet("agyn_env.test", "value"),
					resource.TestCheckResourceAttrSet("agyn_env.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynEnv_update(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynEnvConfig(t, organizationName, agentName, "Terraform acceptance env", "env-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_env.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_env.test", "name", "AGYN_ENV"),
					resource.TestCheckResourceAttr("agyn_env.test", "description", "Terraform acceptance env"),
					resource.TestCheckResourceAttrSet("agyn_env.test", "value"),
					resource.TestCheckResourceAttrSet("agyn_env.test", "id"),
				),
			},
			{
				Config: testAccAgynEnvConfig(t, organizationName, agentName, "Terraform acceptance env updated", "env-value-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_env.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_env.test", "name", "AGYN_ENV"),
					resource.TestCheckResourceAttr("agyn_env.test", "description", "Terraform acceptance env updated"),
					resource.TestCheckResourceAttr("agyn_env.test", "value", "env-value-updated"),
					resource.TestCheckResourceAttrSet("agyn_env.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynEnv_import(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynEnvConfig(t, organizationName, agentName, "Terraform acceptance env", "env-value"),
			},
			{
				ResourceName:      "agyn_env.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynEnvConfig(t *testing.T, organizationName, agentName, description, value string) string {
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

%s

resource "agyn_env" "test" {
	  name        = "AGYN_ENV"
	  description = %q
	  agent_id    = agyn_agent.test.id
	  value       = %q
}
`, testAccProviderConfig(t), organizationName, testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", agentName, "Terraform acceptance agent", "Terraform acceptance role"), description, value)
}
