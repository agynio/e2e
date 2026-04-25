//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynVolumeAttachment_basic(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynVolumeAttachmentConfig(t, organizationName, agentName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_volume_attachment.test", "volume_id", "agyn_volume.primary", "id"),
					resource.TestCheckResourceAttrPair("agyn_volume_attachment.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttrSet("agyn_volume_attachment.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynVolumeAttachment_update(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynVolumeAttachmentConfig(t, organizationName, agentName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_volume_attachment.test", "volume_id", "agyn_volume.primary", "id"),
					resource.TestCheckResourceAttrPair("agyn_volume_attachment.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttrSet("agyn_volume_attachment.test", "id"),
				),
			},
			{
				Config: testAccAgynVolumeAttachmentConfig(t, organizationName, agentName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_volume_attachment.test", "volume_id", "agyn_volume.secondary", "id"),
					resource.TestCheckResourceAttrPair("agyn_volume_attachment.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttrSet("agyn_volume_attachment.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynVolumeAttachment_import(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynVolumeAttachmentConfig(t, organizationName, agentName, false),
			},
			{
				ResourceName:      "agyn_volume_attachment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynVolumeAttachmentConfig(t *testing.T, organizationName, agentName string, useSecondary bool) string {
	volumeRef := "agyn_volume.primary.id"
	if useSecondary {
		volumeRef = "agyn_volume.secondary.id"
	}

	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

%s

resource "agyn_volume" "primary" {
	  organization_id = agyn_organization.test.id
	  persistent  = true
	  mount_path  = "/data"
	  size        = "1Gi"
	  description = "Terraform acceptance volume"
}

resource "agyn_volume" "secondary" {
	  organization_id = agyn_organization.test.id
	  persistent  = true
	  mount_path  = "/data-secondary"
	  size        = "2Gi"
	  description = "Terraform acceptance volume secondary"
}

resource "agyn_volume_attachment" "test" {
	  volume_id = %s
	  agent_id  = agyn_agent.test.id
}
`, testAccProviderConfig(t), organizationName, testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", agentName, "Terraform acceptance agent", "Terraform acceptance role"), volumeRef)
}
