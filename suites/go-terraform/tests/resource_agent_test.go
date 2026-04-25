//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynAgent_basic(t *testing.T) {
	resourceName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	env := testAccEnv(t)
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynAgentConfig(t, organizationName, resourceName, "Terraform acceptance agent", "Terraform acceptance role", nil, []string{"docker"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_agent.test", "name", resourceName),
					resource.TestCheckResourceAttr("agyn_agent.test", "description", "Terraform acceptance agent"),
					resource.TestCheckNoResourceAttr("agyn_agent.test", "nickname"),
					resource.TestCheckResourceAttr("agyn_agent.test", "role", "Terraform acceptance role"),
					resource.TestCheckResourceAttr("agyn_agent.test", "capabilities.#", "1"),
					resource.TestCheckResourceAttr("agyn_agent.test", "capabilities.0", "docker"),
					resource.TestCheckResourceAttr("agyn_agent.test", "init_image", env.AgentInitImage),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "organization_id"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "model"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynAgent_update(t *testing.T) {
	resourceName := acctest.RandomWithPrefix("tf-acc-agent")
	updatedName := resourceName + "-updated"
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	nickname := "tf-acc-nickname"
	updatedNickname := "tf-acc-nickname-updated"
	env := testAccEnv(t)
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynAgentConfig(t, organizationName, resourceName, "Terraform acceptance agent", "Terraform acceptance role", &nickname, []string{"docker"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_agent.test", "name", resourceName),
					resource.TestCheckResourceAttr("agyn_agent.test", "description", "Terraform acceptance agent"),
					resource.TestCheckResourceAttr("agyn_agent.test", "nickname", nickname),
					resource.TestCheckResourceAttr("agyn_agent.test", "role", "Terraform acceptance role"),
					resource.TestCheckResourceAttr("agyn_agent.test", "capabilities.#", "1"),
					resource.TestCheckResourceAttr("agyn_agent.test", "capabilities.0", "docker"),
					resource.TestCheckResourceAttr("agyn_agent.test", "init_image", env.AgentInitImage),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "organization_id"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "model"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "id"),
				),
			},
			{
				Config: testAccAgynAgentConfig(t, organizationName, updatedName, "Terraform acceptance agent updated", "Terraform acceptance role updated", &updatedNickname, []string{"docker", "gpu"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_agent.test", "name", updatedName),
					resource.TestCheckResourceAttr("agyn_agent.test", "description", "Terraform acceptance agent updated"),
					resource.TestCheckResourceAttr("agyn_agent.test", "nickname", updatedNickname),
					resource.TestCheckResourceAttr("agyn_agent.test", "role", "Terraform acceptance role updated"),
					resource.TestCheckResourceAttr("agyn_agent.test", "capabilities.#", "2"),
					resource.TestCheckResourceAttr("agyn_agent.test", "init_image", env.AgentInitImage),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "organization_id"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "model"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "image"),
					resource.TestCheckResourceAttrSet("agyn_agent.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynAgent_import(t *testing.T) {
	resourceName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	nickname := "tf-acc-import-nickname"
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynAgentConfig(t, organizationName, resourceName, "Terraform acceptance agent", "Terraform acceptance role", &nickname, []string{"docker"}),
			},
			{
				ResourceName:      "agyn_agent.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAgynAgent_expectError(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccAgynAgentInvalidConfig(t, organizationName),
				ExpectError: regexp.MustCompile("Invalid JSON"),
			},
		},
	})
}

func TestAccAgynAgent_invalidNickname(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccAgynAgentInvalidNicknameConfig(t, organizationName),
				ExpectError: regexp.MustCompile("must contain only lowercase letters"),
			},
		},
	})
}

func testAccAgynAgentConfig(t *testing.T, organizationName, title, description, role string, nickname *string, capabilities []string) string {
	env := testAccEnv(t)

	nicknameLine := ""
	if nickname != nil {
		nicknameLine = fmt.Sprintf("\n\t\tnickname    = %q", *nickname)
	}
	capabilityLine := formatCapabilitiesLine(capabilities, "\t\t")
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

resource "agyn_agent" "test" {
	  organization_id = agyn_organization.test.id
	  name        = %q
	  description = %q
	  role        = %q
	  model       = %q
	  image       = %q
	  init_image  = %q
%s
%s
}
`, testAccProviderConfig(t), organizationName, title, description, role, env.ModelID, env.AgentImage, env.AgentInitImage, nicknameLine, capabilityLine)
}

func testAccAgynAgentInvalidConfig(t *testing.T, organizationName string) string {
	env := testAccEnv(t)

	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

resource "agyn_agent" "test" {
	  organization_id = agyn_organization.test.id
	  name          = "invalid"
	  role          = "invalid"
	  model         = %q
	  image         = %q
	  init_image    = %q
	  configuration = "{invalid"
}
`, testAccProviderConfig(t), organizationName, env.ModelID, env.AgentImage, env.AgentInitImage)
}

func testAccAgynAgentInvalidNicknameConfig(t *testing.T, organizationName string) string {
	env := testAccEnv(t)

	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

resource "agyn_agent" "test" {
	  organization_id = agyn_organization.test.id
	  name          = "invalid"
	  nickname      = "Invalid Nick"
	  role          = "invalid"
	  model         = %q
	  image         = %q
	  init_image    = %q
}
`, testAccProviderConfig(t), organizationName, env.ModelID, env.AgentImage, env.AgentInitImage)
}
