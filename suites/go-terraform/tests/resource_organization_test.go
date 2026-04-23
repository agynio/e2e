//go:build e2e && svc_gateway

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynOrganization_basic(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynOrganizationConfig(t, organizationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_organization.test", "name", organizationName),
					resource.TestCheckResourceAttrSet("agyn_organization.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynOrganization_update(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	updatedName := organizationName + "-updated"
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynOrganizationConfig(t, organizationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_organization.test", "name", organizationName),
					resource.TestCheckResourceAttrSet("agyn_organization.test", "id"),
				),
			},
			{
				Config: testAccAgynOrganizationConfig(t, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_organization.test", "name", updatedName),
					resource.TestCheckResourceAttrSet("agyn_organization.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynOrganization_import(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynOrganizationConfig(t, organizationName),
			},
			{
				ResourceName:      "agyn_organization.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynOrganizationConfig(t *testing.T, name string) string {
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}
`, testAccProviderConfig(t), name)
}
