package provider

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccArgoCDProjectToken(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
resource "argocd_project_token" "this" {
	project     = "myproject1"
	role        = "test-role1234"
	description = "test"
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_project_token.this", "project", "myproject1"),
					resource.TestCheckResourceAttr("argocd_project_token.this", "role", "test-role1234"),
					resource.TestCheckResourceAttr("argocd_project_token.this", "description", "test"),
					resource.TestCheckNoResourceAttr("argocd_project_token.this", "expires_at"),
					resource.TestCheckNoResourceAttr("argocd_project_token.this", "expires_in"),
					resource.TestCheckResourceAttrSet("argocd_project_token.this", "id"),
					resource.TestCheckResourceAttrSet("argocd_project_token.this", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_project_token.this", "jwt"),
					resource.TestCheckNoResourceAttr("argocd_project_token.this", "renew_after"),
				),
			},
			// Update
			{
				Config: `
resource "argocd_project_token" "this" {
	project     = "myproject1"
	role        = "test-role1234"
	description = "test"
	expires_in  = "24h"
	renew_after = "12h"
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_project_token.this", "project", "myproject1"),
					resource.TestCheckResourceAttr("argocd_project_token.this", "role", "test-role1234"),
					resource.TestCheckResourceAttr("argocd_project_token.this", "description", "test"),
					resource.TestCheckResourceAttrSet("argocd_project_token.this", "expires_at"),
					resource.TestCheckResourceAttr("argocd_project_token.this", "expires_in", "24h"),
					resource.TestCheckResourceAttrSet("argocd_project_token.this", "id"),
					resource.TestCheckResourceAttrSet("argocd_project_token.this", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_project_token.this", "jwt"),
					resource.TestCheckResourceAttr("argocd_project_token.this", "renew_after", "12h"),
				),
			},
		},
	})
}

func TestAccArgoCDProjectToken_Expires(t *testing.T) {
	expiresInSeconds := 15
	config := fmt.Sprintf(`
resource "argocd_project_token" "expires" {
	project    = "myproject1"
	role       = "test-role1234"
	expires_in = "%ds"
}
	`, expiresInSeconds)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_project_token.expires", "project", "myproject1"),
					resource.TestCheckResourceAttr("argocd_project_token.expires", "role", "test-role1234"),
					testCheckTokenExpiresAt("argocd_project_token.expires", int64(expiresInSeconds)),
					resource.TestCheckResourceAttr("argocd_project_token.expires", "expires_in", fmt.Sprintf("%ds", expiresInSeconds)),
					resource.TestCheckResourceAttrSet("argocd_project_token.expires", "id"),
					resource.TestCheckResourceAttrSet("argocd_project_token.expires", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_project_token.expires", "jwt"),
					resource.TestCheckNoResourceAttr("argocd_project_token.expires", "renew_after"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testDelay(expiresInSeconds + 1),
				),
				ExpectNonEmptyPlan: true, // token should be recreated when refreshed at end of step due to delay above
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTokenExpiresAt("argocd_project_token.expires", int64(expiresInSeconds)),
				),
			},
		},
	})
}

func TestAccArgoCDProjectToken_Multiple(t *testing.T) {
	count := 3 + rand.Intn(7)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "argocd_project_token" "multiple1a" {
	count = %[1]d
	project = "myproject1"
	role    = "test-role1234"
}

resource "argocd_project_token" "multiple1b" {
	count = %[1]d
	project = "myproject1"
	role    = "test-role4321"
}

resource "argocd_project_token" "multiple2a" {
	count   = %[1]d
	project = "myproject2"
	role    = "test-role1234"
}

resource "argocd_project_token" "multiple2b" {
	count   = %[1]d
	project = "myproject2"
	role    = "test-role4321"
}
				`, count),
				Check: resource.ComposeTestCheckFunc(
					testTokenIssuedAtSet("argocd_project_token.multiple1a", count),
					testTokenIssuedAtSet("argocd_project_token.multiple1b", count),
					testTokenIssuedAtSet("argocd_project_token.multiple2a", count),
					testTokenIssuedAtSet("argocd_project_token.multiple2b", count),
				),
			},
		},
	})
}

func TestAccArgoCDProjectToken_RenewAfter(t *testing.T) {
	renewAfterSeconds := 15
	config := fmt.Sprintf(`
resource "argocd_project_token" "renew_after" {
	project     = "myproject1"
	description = "auto-renewing long-lived token"
	role        = "test-role1234"
	renew_after = "%ds"
}
	`, renewAfterSeconds)

	// Note: not running in parallel as this is a time sensitive test case
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_project_token.renew_after", "project", "myproject1"),
					resource.TestCheckResourceAttr("argocd_project_token.renew_after", "description", "auto-renewing long-lived token"),
					resource.TestCheckResourceAttr("argocd_project_token.renew_after", "role", "test-role1234"),
					resource.TestCheckNoResourceAttr("argocd_project_token.renew_after", "expires_at"),
					resource.TestCheckNoResourceAttr("argocd_project_token.renew_after", "expires_in"),
					resource.TestCheckResourceAttrSet("argocd_project_token.renew_after", "id"),
					resource.TestCheckResourceAttrSet("argocd_project_token.renew_after", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_project_token.renew_after", "jwt"),
					resource.TestCheckResourceAttr("argocd_project_token.renew_after", "renew_after", fmt.Sprintf("%ds", renewAfterSeconds)),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testDelay(renewAfterSeconds + 1),
				),
				ExpectNonEmptyPlan: true, // token should be recreated when refreshed at end of step due to delay above
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_project_token.renew_after", "renew_after", fmt.Sprintf("%ds", renewAfterSeconds)),
				),
			},
		},
	})
}

func TestAccArgoCDProjectToken_UpgradeFromV5(t *testing.T) {
	config := `
resource "argocd_project_token" "upgrade" {
	project = "myproject1"
	role    = "test-role1234"
	expires_in  = "24h"
	renew_after = "12h"
}`

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: providerV5(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_project_token.upgrade", "project", "myproject1"),
					resource.TestCheckResourceAttr("argocd_project_token.upgrade", "role", "test-role1234"),
					resource.TestCheckResourceAttrSet("argocd_project_token.upgrade", "expires_at"),
					resource.TestCheckResourceAttr("argocd_project_token.upgrade", "expires_in", "24h"),
					resource.TestCheckResourceAttrSet("argocd_project_token.upgrade", "id"),
					resource.TestCheckResourceAttrSet("argocd_project_token.upgrade", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_project_token.upgrade", "jwt"),
					resource.TestCheckResourceAttr("argocd_project_token.upgrade", "renew_after", "12h"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}
