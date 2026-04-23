//go:build e2e && svc_gateway

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynSkill_basic(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	skillName := acctest.RandomWithPrefix("tf-acc-skill")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynSkillConfig(t, organizationName, agentName, skillName, "Terraform acceptance skill", "run"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_skill.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_skill.test", "name", skillName),
					resource.TestCheckResourceAttr("agyn_skill.test", "body", "run"),
					resource.TestCheckResourceAttr("agyn_skill.test", "description", "Terraform acceptance skill"),
					resource.TestCheckResourceAttrSet("agyn_skill.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynSkill_update(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	skillName := acctest.RandomWithPrefix("tf-acc-skill")
	updatedName := skillName + "-updated"
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynSkillConfig(t, organizationName, agentName, skillName, "Terraform acceptance skill", "run"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_skill.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_skill.test", "name", skillName),
					resource.TestCheckResourceAttr("agyn_skill.test", "body", "run"),
					resource.TestCheckResourceAttr("agyn_skill.test", "description", "Terraform acceptance skill"),
					resource.TestCheckResourceAttrSet("agyn_skill.test", "id"),
				),
			},
			{
				Config: testAccAgynSkillConfig(t, organizationName, agentName, updatedName, "Terraform acceptance skill updated", "run updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_skill.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttr("agyn_skill.test", "name", updatedName),
					resource.TestCheckResourceAttr("agyn_skill.test", "body", "run updated"),
					resource.TestCheckResourceAttr("agyn_skill.test", "description", "Terraform acceptance skill updated"),
					resource.TestCheckResourceAttrSet("agyn_skill.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynSkill_import(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	skillName := acctest.RandomWithPrefix("tf-acc-skill")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynSkillConfig(t, organizationName, agentName, skillName, "Terraform acceptance skill", "run"),
			},
			{
				ResourceName:      "agyn_skill.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynSkillConfig(t *testing.T, organizationName, agentName, skillName, description, body string) string {
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

%s

resource "agyn_skill" "test" {
	  agent_id    = agyn_agent.test.id
	  name        = %q
	  body        = %q
	  description = %q
}
`, testAccProviderConfig(t), organizationName, testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", agentName, "Terraform acceptance agent", "Terraform acceptance role"), skillName, body, description)
}
