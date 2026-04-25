//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynApp_basic(t *testing.T) {
	slug := acctest.RandomWithPrefix("tf-acc-app")
	name := fmt.Sprintf("Terraform acceptance app %s", slug)
	organizationID := requireOrganizationID(t)
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccAppPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynAppConfig(t, slug, name, "Terraform acceptance app", "https://example.com/icon.png", organizationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_app.test", "organization_id", organizationID),
					resource.TestCheckResourceAttr("agyn_app.test", "slug", slug),
					resource.TestCheckResourceAttr("agyn_app.test", "name", name),
					resource.TestCheckResourceAttr("agyn_app.test", "description", "Terraform acceptance app"),
					resource.TestCheckResourceAttr("agyn_app.test", "icon", "https://example.com/icon.png"),
					resource.TestCheckResourceAttr("agyn_app.test", "visibility", "internal"),
					resource.TestCheckResourceAttr("agyn_app.test", "permissions.#", "1"),
					resource.TestCheckResourceAttr("agyn_app.test", "permissions.0", "thread:write"),
					resource.TestCheckResourceAttrSet("agyn_app.test", "identity_id"),
					resource.TestCheckResourceAttrSet("agyn_app.test", "service_token"),
					resource.TestCheckResourceAttrSet("agyn_app.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynApp_import(t *testing.T) {
	slug := acctest.RandomWithPrefix("tf-acc-app")
	name := fmt.Sprintf("Terraform acceptance app %s", slug)
	organizationID := requireOrganizationID(t)
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccAppPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynAppConfig(t, slug, name, "Terraform acceptance app", "https://example.com/icon.png", organizationID),
			},
			{
				ResourceName:            "agyn_app.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"service_token"},
			},
		},
	})
}

func testAccAppPreCheck(t *testing.T) {
	testAccPreCheck(t)
	requireAPIToken(t)
	requireOrganizationID(t)
}

func testAccAgynAppConfig(t *testing.T, slug, name, description, icon, organizationID string) string {
	return fmt.Sprintf(`
%s

resource "agyn_app" "test" {
	  organization_id = %q
	  slug            = %q
	  name            = %q
	  description     = %q
	  icon            = %q
	  visibility      = "internal"
	  permissions     = ["thread:write"]
}
`, testAccProviderConfig(t), organizationID, slug, name, description, icon)
}
