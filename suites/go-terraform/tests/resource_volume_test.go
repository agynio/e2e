//go:build e2e && svc_gateway

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynVolume_basic(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynVolumeConfig(t, organizationName, "Terraform acceptance volume", "/data", "1Gi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_volume.test", "persistent", "true"),
					resource.TestCheckResourceAttr("agyn_volume.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("agyn_volume.test", "size", "1Gi"),
					resource.TestCheckResourceAttr("agyn_volume.test", "description", "Terraform acceptance volume"),
					resource.TestCheckResourceAttrSet("agyn_volume.test", "organization_id"),
					resource.TestCheckResourceAttrSet("agyn_volume.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynVolume_update(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynVolumeConfig(t, organizationName, "Terraform acceptance volume", "/data", "1Gi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_volume.test", "persistent", "true"),
					resource.TestCheckResourceAttr("agyn_volume.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("agyn_volume.test", "size", "1Gi"),
					resource.TestCheckResourceAttr("agyn_volume.test", "description", "Terraform acceptance volume"),
					resource.TestCheckResourceAttrSet("agyn_volume.test", "organization_id"),
					resource.TestCheckResourceAttrSet("agyn_volume.test", "id"),
				),
			},
			{
				Config: testAccAgynVolumeConfig(t, organizationName, "Terraform acceptance volume updated", "/data-updated", "2Gi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_volume.test", "persistent", "true"),
					resource.TestCheckResourceAttr("agyn_volume.test", "mount_path", "/data-updated"),
					resource.TestCheckResourceAttr("agyn_volume.test", "size", "2Gi"),
					resource.TestCheckResourceAttr("agyn_volume.test", "description", "Terraform acceptance volume updated"),
					resource.TestCheckResourceAttrSet("agyn_volume.test", "organization_id"),
					resource.TestCheckResourceAttrSet("agyn_volume.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynVolume_import(t *testing.T) {
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynVolumeConfig(t, organizationName, "Terraform acceptance volume", "/data", "1Gi"),
			},
			{
				ResourceName:            "agyn_volume.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"organization_id"},
			},
		},
	})
}

func testAccAgynVolumeConfig(t *testing.T, organizationName, description, mountPath, size string) string {
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

resource "agyn_volume" "test" {
	  organization_id = agyn_organization.test.id
	  persistent  = true
	  mount_path  = %q
	  size        = %q
	  description = %q
}
`, testAccProviderConfig(t), organizationName, mountPath, size, description)
}
