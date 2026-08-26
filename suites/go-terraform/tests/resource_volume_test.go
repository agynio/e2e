//go:build e2e && (svc_gateway || tf_provider_agyn)

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
				Config: testAccAgynVolumeConfig(t, organizationName, "tf-acc-volume", "/data", "1Gi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_volume.test", "name", "tf-acc-volume"),
					resource.TestCheckResourceAttr("agyn_volume.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("agyn_volume.test", "size", "1Gi"),
					resource.TestCheckResourceAttrPair("agyn_volume.test", "environment_id", "agyn_environment.test", "id"),
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
				Config: testAccAgynVolumeConfig(t, organizationName, "tf-acc-volume", "/data", "1Gi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_volume.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("agyn_volume.test", "size", "1Gi"),
				),
			},
			{
				Config: testAccAgynVolumeConfig(t, organizationName, "tf-acc-volume", "/data-updated", "2Gi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_volume.test", "name", "tf-acc-volume"),
					resource.TestCheckResourceAttr("agyn_volume.test", "mount_path", "/data-updated"),
					resource.TestCheckResourceAttr("agyn_volume.test", "size", "2Gi"),
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
				Config: testAccAgynVolumeConfig(t, organizationName, "tf-acc-volume", "/data", "1Gi"),
			},
			{
				ResourceName:      "agyn_volume.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// A volume belongs to an environment (or an MCP), so the environment the agent
// fixtures already build is what this one declares itself against.
func testAccAgynVolumeConfig(t *testing.T, organizationName, name, mountPath, size string) string {
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

%s

resource "agyn_volume" "test" {
	  environment_id = agyn_environment.test.id
	  name           = %q
	  mount_path     = %q
	  size           = %q
}
`, testAccProviderConfig(t), organizationName,
		testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", "tf-acc-volume-agent", "Terraform acceptance agent", "Terraform acceptance role"),
		name, mountPath, size)
}
