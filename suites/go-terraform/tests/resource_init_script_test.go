//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynInitScript_basic(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynInitScriptConfig(t, organizationName, agentName, "Terraform acceptance init script", "echo 'Hello, World!'"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_init_script.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_init_script.test", "description", "Terraform acceptance init script"),
					resource.TestCheckResourceAttr("agyn_init_script.test", "script", "echo 'Hello, World!'"),
					resource.TestCheckResourceAttrSet("agyn_init_script.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynInitScript_update(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynInitScriptConfig(t, organizationName, agentName, "Terraform acceptance init script", "echo 'Hello, World!'"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_init_script.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_init_script.test", "description", "Terraform acceptance init script"),
					resource.TestCheckResourceAttr("agyn_init_script.test", "script", "echo 'Hello, World!'"),
					resource.TestCheckResourceAttrSet("agyn_init_script.test", "id"),
				),
			},
			{
				Config: testAccAgynInitScriptConfig(t, organizationName, agentName, "Terraform acceptance init script updated", "echo 'Hello, Updated World!'"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_init_script.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_init_script.test", "description", "Terraform acceptance init script updated"),
					resource.TestCheckResourceAttr("agyn_init_script.test", "script", "echo 'Hello, Updated World!'"),
					resource.TestCheckResourceAttrSet("agyn_init_script.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynInitScript_import(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynInitScriptConfig(t, organizationName, agentName, "Terraform acceptance init script", "echo 'Hello, World!'"),
			},
			{
				ResourceName:      "agyn_init_script.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynInitScriptConfig(t *testing.T, organizationName, agentName, description, script string) string {
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

%s

resource "agyn_init_script" "test" {
	  agent_id    = agyn_agent.test.id
	  description = %q
	  script      = %q
}
`, testAccProviderConfig(t), organizationName, testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", agentName, "Terraform acceptance agent", "Terraform acceptance role"), description, script)
}
