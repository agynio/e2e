//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAgynUser_basic(t *testing.T) {
	oidcSubject := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-user"))
	name := "Terraform acceptance user"
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccUserPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynUserConfig(t, oidcSubject, name, "", "", "admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_user.test", "oidc_subject", oidcSubject),
					resource.TestCheckResourceAttr("agyn_user.test", "name", name),
					resource.TestCheckResourceAttr("agyn_user.test", "cluster_role", "admin"),
					resource.TestCheckResourceAttrSet("agyn_user.test", "identity_id"),
				),
			},
		},
	})
}

func TestAccAgynUser_update(t *testing.T) {
	oidcSubject := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-user"))
	name := "Terraform acceptance user"
	updatedName := "Terraform acceptance user updated"
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccUserPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynUserConfig(t, oidcSubject, name, "", "", "admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_user.test", "name", name),
					resource.TestCheckResourceAttr("agyn_user.test", "cluster_role", "admin"),
					resource.TestCheckResourceAttrSet("agyn_user.test", "identity_id"),
				),
			},
			{
				Config: testAccAgynUserConfig(t, oidcSubject, updatedName, "", "", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("agyn_user.test", "name", updatedName),
					resource.TestCheckResourceAttrSet("agyn_user.test", "identity_id"),
				),
			},
		},
	})
}

func TestAccAgynUser_import(t *testing.T) {
	oidcSubject := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-user"))
	name := "Terraform acceptance user"
	resource.Test(t, resource.TestCase{
		ExternalProviders: testAccExternalProviders(),
		PreCheck:          func() { testAccUserPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccAgynUserConfig(t, oidcSubject, name, "", "", "admin"),
			},
			{
				ResourceName:                         "agyn_user.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    testAccUserImportID,
				ImportStateVerifyIdentifierAttribute: "identity_id",
			},
		},
	})
}

func testAccUserImportID(state *terraform.State) (string, error) {
	resourceState, ok := state.RootModule().Resources["agyn_user.test"]
	if !ok {
		return "", fmt.Errorf("agyn_user.test not found in state")
	}
	identityID := resourceState.Primary.Attributes["identity_id"]
	if identityID == "" {
		return "", fmt.Errorf("identity_id missing from state")
	}
	return identityID, nil
}

func testAccUserPreCheck(t *testing.T) {
	testAccPreCheck(t)
	requireAPIToken(t)
}

func testAccAgynUserConfig(t *testing.T, oidcSubject, name, photoURL, nickname, clusterRole string) string {
	nameLine := ""
	if name != "" {
		nameLine = fmt.Sprintf("\n\t  name         = %q", name)
	}
	photoLine := ""
	if photoURL != "" {
		photoLine = fmt.Sprintf("\n\t  photo_url    = %q", photoURL)
	}
	nicknameLine := ""
	if nickname != "" {
		nicknameLine = fmt.Sprintf("\n\t  nickname     = %q", nickname)
	}
	clusterLine := ""
	if clusterRole != "" {
		clusterLine = fmt.Sprintf("\n\t  cluster_role = %q", clusterRole)
	}

	return fmt.Sprintf(`
%s

resource "agyn_user" "test" {
	  oidc_subject = %q%s%s%s%s
}
`, testAccProviderConfig(t), oidcSubject, nameLine, photoLine, nicknameLine, clusterLine)
}
