//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynHook_basic(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynHookConfig(t, organizationName, agentName, "Terraform acceptance hook", "agent.started", "handler"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_hook.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_hook.test", "event", "agent.started"),
					resource.TestCheckResourceAttr("agyn_hook.test", "function", "handler"),
					resource.TestCheckResourceAttr("agyn_hook.test", "description", "Terraform acceptance hook"),
					resource.TestCheckResourceAttrSet("agyn_hook.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_hook.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynHook_update(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynHookConfig(t, organizationName, agentName, "Terraform acceptance hook", "agent.started", "handler"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_hook.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_hook.test", "event", "agent.started"),
					resource.TestCheckResourceAttr("agyn_hook.test", "function", "handler"),
					resource.TestCheckResourceAttr("agyn_hook.test", "description", "Terraform acceptance hook"),
					resource.TestCheckResourceAttrSet("agyn_hook.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_hook.test", "id"),
				),
			},
			{
				Config: testAccAgynHookConfig(t, organizationName, agentName, "Terraform acceptance hook updated", "agent.started", "handler"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_hook.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_hook.test", "event", "agent.started"),
					resource.TestCheckResourceAttr("agyn_hook.test", "function", "handler"),
					resource.TestCheckResourceAttr("agyn_hook.test", "description", "Terraform acceptance hook updated"),
					resource.TestCheckResourceAttrSet("agyn_hook.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_hook.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynHook_import(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynHookConfig(t, organizationName, agentName, "Terraform acceptance hook", "agent.started", "handler"),
			},
			{
				ResourceName:      "agyn_hook.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynHookConfig(t *testing.T, organizationName, agentName, description, event, function string) string {
	env := testAccEnv(t)

	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

%s

resource "agyn_hook" "test" {
	  agent_id    = agyn_agent.test.id
	  event       = %q
	  function    = %q
	  image       = %q
	  description = %q
}
`, testAccProviderConfig(t), organizationName, testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", agentName, "Terraform acceptance agent", "Terraform acceptance role"), event, function, env.AgentImage, description)
}
