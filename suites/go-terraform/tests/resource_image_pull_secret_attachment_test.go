//go:build e2e && svc_gateway

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynImagePullSecretAttachment_basic(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	secretDescription := acctest.RandomWithPrefix("tf-acc-secret")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynImagePullSecretAttachmentConfig(t, organizationName, agentName, secretDescription, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret_attachment.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret_attachment.test", "image_pull_secret_id", "agyn_image_pull_secret.primary", "id"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret_attachment.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynImagePullSecretAttachment_update(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	secretDescription := acctest.RandomWithPrefix("tf-acc-secret")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynImagePullSecretAttachmentConfig(t, organizationName, agentName, secretDescription, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret_attachment.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret_attachment.test", "image_pull_secret_id", "agyn_image_pull_secret.primary", "id"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret_attachment.test", "id"),
				),
			},
			{
				Config: testAccAgynImagePullSecretAttachmentConfig(t, organizationName, agentName, secretDescription, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret_attachment.test", "agent_id", "agyn_agent.test", "id"),
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret_attachment.test", "image_pull_secret_id", "agyn_image_pull_secret.secondary", "id"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret_attachment.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynImagePullSecretAttachment_import(t *testing.T) {
	agentName := acctest.RandomWithPrefix("tf-acc-agent")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	secretDescription := acctest.RandomWithPrefix("tf-acc-secret")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynImagePullSecretAttachmentConfig(t, organizationName, agentName, secretDescription, false),
			},
			{
				ResourceName:      "agyn_image_pull_secret_attachment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAgynImagePullSecretAttachmentConfig(t *testing.T, organizationName, agentName, secretDescription string, useSecondary bool) string {
	secretRef := "agyn_image_pull_secret.primary.id"
	secondaryDescription := secretDescription + "-secondary"
	if useSecondary {
		secretRef = "agyn_image_pull_secret.secondary.id"
	}

	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

resource "agyn_image_pull_secret" "primary" {
	  organization_id = agyn_organization.test.id
	  description     = %q
	  registry        = "registry.example.com"
	  username        = "registry-user"
	  password        = "registry-password"
}

resource "agyn_image_pull_secret" "secondary" {
	  organization_id = agyn_organization.test.id
	  description     = %q
	  registry        = "registry-secondary.example.com"
	  username        = "registry-user-secondary"
	  password        = "registry-password-secondary"
}

%s

resource "agyn_image_pull_secret_attachment" "test" {
	  agent_id             = agyn_agent.test.id
	  image_pull_secret_id = %s
}
`, testAccProviderConfig(t), organizationName, secretDescription, secondaryDescription, testAccAgynAgentResourceBlock(t, "agyn_organization.test.id", agentName, "Terraform acceptance agent", "Terraform acceptance role"), secretRef)
}
