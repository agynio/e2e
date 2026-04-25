//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgynImagePullSecret_basic(t *testing.T) {
	secretDescription := acctest.RandomWithPrefix("tf-acc-secret")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynImagePullSecretConfig(t, organizationName, secretDescription, "registry.example.com", "registry-user", "registry-password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret.test", "organization_id", "agyn_organization.test", "id"),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "description", secretDescription),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "registry", "registry.example.com"),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "username", "registry-user"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret.test", "password"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynImagePullSecret_update(t *testing.T) {
	secretDescription := acctest.RandomWithPrefix("tf-acc-secret")
	updatedSecretDescription := secretDescription + "-updated"
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynImagePullSecretConfig(t, organizationName, secretDescription, "registry.example.com", "registry-user", "registry-password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret.test", "organization_id", "agyn_organization.test", "id"),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "description", secretDescription),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "registry", "registry.example.com"),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "username", "registry-user"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret.test", "password"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret.test", "id"),
				),
			},
			{
				Config: testAccAgynImagePullSecretConfig(t, organizationName, updatedSecretDescription, "registry-updated.example.com", "registry-user-updated", "registry-password-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("agyn_image_pull_secret.test", "organization_id", "agyn_organization.test", "id"),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "description", updatedSecretDescription),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "registry", "registry-updated.example.com"),
					resource.TestCheckResourceAttr("agyn_image_pull_secret.test", "username", "registry-user-updated"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret.test", "password"),
					resource.TestCheckResourceAttrSet("agyn_image_pull_secret.test", "id"),
				),
			},
		},
	})
}

func TestAccAgynImagePullSecret_import(t *testing.T) {
	secretDescription := acctest.RandomWithPrefix("tf-acc-secret")
	organizationName := acctest.RandomWithPrefix("tf-acc-org")
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccOrganizationPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynImagePullSecretConfig(t, organizationName, secretDescription, "registry.example.com", "registry-user", "registry-password"),
			},
			{
				ResourceName:            "agyn_image_pull_secret.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"organization_id", "password"},
			},
		},
	})
}

func testAccAgynImagePullSecretConfig(t *testing.T, organizationName, description, registry, username, password string) string {
	return fmt.Sprintf(`
%s

resource "agyn_organization" "test" {
	  name = %q
}

resource "agyn_image_pull_secret" "test" {
	  organization_id = agyn_organization.test.id
	  description     = %q
	  registry        = %q
	  username        = %q
	  password        = %q
}
`, testAccProviderConfig(t), organizationName, description, registry, username, password)
}
