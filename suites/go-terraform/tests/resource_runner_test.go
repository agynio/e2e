//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccAgynRunner_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-runner")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccRunnerPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynRunnerConfig(t, name, []string{"docker"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_runner.test", "name", name),
					resource.TestCheckResourceAttr("agyn_runner.test", "capabilities.#", "1"),
					resource.TestCheckResourceAttr("agyn_runner.test", "capabilities.0", "docker"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "identity_id"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "service_token"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynRunner_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-runner")
	updatedName := name + "-updated"
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccRunnerPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynRunnerConfigWithLabels(t, name, "test", "infra", []string{"docker"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_runner.test", "name", name),
					resource.TestCheckResourceAttr("agyn_runner.test", "labels.%", "2"),
					resource.TestCheckResourceAttr("agyn_runner.test", "labels.environment", "test"),
					resource.TestCheckResourceAttr("agyn_runner.test", "labels.team", "infra"),
					resource.TestCheckResourceAttr("agyn_runner.test", "capabilities.#", "1"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "identity_id"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "service_token"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "id"),
				),
			},
			{
				Config: testAccAgynRunnerConfigWithLabels(t, updatedName, "prod", "platform", []string{"docker", "gpu"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_runner.test", "name", updatedName),
					resource.TestCheckResourceAttr("agyn_runner.test", "labels.%", "2"),
					resource.TestCheckResourceAttr("agyn_runner.test", "labels.environment", "prod"),
					resource.TestCheckResourceAttr("agyn_runner.test", "labels.team", "platform"),
					resource.TestCheckResourceAttr("agyn_runner.test", "capabilities.#", "2"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "identity_id"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "service_token"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynRunner_organizationIDRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-runner")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	updatedOrganizationName := organizationName + "-updated"
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccRunnerOrgPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynRunnerConfigWithOrganizationResource(t, name, "primary", organizationName, []string{"docker"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_runner.test", "name", name),
					resource.TestCheckResourceAttrPair("agyn_runner.test", "organization_id", "agyn_organization.primary", "id"),
					resource.TestCheckResourceAttr("agyn_runner.test", "capabilities.#", "1"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "identity_id"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "service_token"),
					resource.TestCheckResourceAttrSet("agyn_runner.test", "id"),
				),
			},
			{
				Config: testAccAgynRunnerConfigWithOrganizationResource(t, name, "secondary", updatedOrganizationName, []string{"docker"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("agyn_runner.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestAccAgynRunner_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-runner")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccRunnerPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynRunnerConfig(t, name, []string{"docker"}),
			},
			{
				ResourceName:            "agyn_runner.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"service_token"},
			},
		},
	})
}

func testAccRunnerPreCheck(t *testing.T) {
	testAccPreCheck(t)
	requireAPIToken(t)
}

func testAccRunnerOrgPreCheck(t *testing.T) {
	testAccRunnerPreCheck(t)
}

func testAccAgynRunnerConfig(t *testing.T, name string, capabilities []string) string {
	capabilityLine := formatCapabilitiesLine(capabilities, "\t  ")
	return fmt.Sprintf(`
%s

resource "agyn_runner" "test" {
	  name = %q
%s
}
`, testAccProviderConfig(t), name, capabilityLine)
}

func testAccAgynRunnerConfigWithLabels(t *testing.T, name, environment, team string, capabilities []string) string {
	capabilityLine := formatCapabilitiesLine(capabilities, "\t  ")
	return fmt.Sprintf(`
%s

resource "agyn_runner" "test" {
	  name = %q
	  labels = {
			environment = %q
			team        = %q
	  }
%s
}
`, testAccProviderConfig(t), name, environment, team, capabilityLine)
}

func testAccAgynRunnerConfigWithOrganizationResource(t *testing.T, name, orgResourceName, organizationName string, capabilities []string) string {
	capabilityLine := formatCapabilitiesLine(capabilities, "\t  ")
	return fmt.Sprintf(`
%s

resource "agyn_organization" "%s" {
	  name = %q
}

resource "agyn_runner" "test" {
	  name            = %q
	  organization_id = agyn_organization.%s.id
%s
}
`, testAccProviderConfig(t), orgResourceName, organizationName, name, orgResourceName, capabilityLine)
}
