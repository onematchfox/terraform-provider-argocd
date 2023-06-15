package provider

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccArgoCDAccountToken(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
resource "argocd_account_token" "this" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("argocd_account_token.this", "account"),
					resource.TestCheckNoResourceAttr("argocd_account_token.this", "expires_at"),
					resource.TestCheckNoResourceAttr("argocd_account_token.this", "expires_in"),
					resource.TestCheckResourceAttrSet("argocd_account_token.this", "id"),
					resource.TestCheckResourceAttrSet("argocd_account_token.this", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_account_token.this", "jwt"),
					resource.TestCheckNoResourceAttr("argocd_account_token.this", "renew_after"),
				),
			},
			// Update
			{
				Config: `
resource "argocd_account_token" "this" {
	expires_in  = "24h"
	renew_after = "12h"
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("argocd_account_token.this", "account"),
					resource.TestCheckResourceAttrSet("argocd_account_token.this", "expires_at"),
					resource.TestCheckResourceAttr("argocd_account_token.this", "expires_in", "24h"),
					resource.TestCheckResourceAttrSet("argocd_account_token.this", "id"),
					resource.TestCheckResourceAttrSet("argocd_account_token.this", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_account_token.this", "jwt"),
					resource.TestCheckResourceAttr("argocd_account_token.this", "renew_after", "12h"),
				),
			},
		},
	})
}

func TestAccArgoCDAccountToken_ExplicitAccount(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "argocd_account_token" "test" {
	account = "test"
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_account_token.test", "account", "test"),
					resource.TestCheckNoResourceAttr("argocd_account_token.test", "expires_at"),
					resource.TestCheckNoResourceAttr("argocd_account_token.test", "expires_in"),
					resource.TestCheckResourceAttrSet("argocd_account_token.test", "id"),
					resource.TestCheckResourceAttrSet("argocd_account_token.test", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_account_token.test", "jwt"),
					resource.TestCheckNoResourceAttr("argocd_account_token.test", "renew_after"),
				),
			},
		},
	})
}

func TestAccArgoCDAccountToken_Expires(t *testing.T) {
	expiresInSeconds := 15
	config := fmt.Sprintf(`
resource "argocd_account_token" "expires" {
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
					resource.TestCheckNoResourceAttr("argocd_account_token.expires", "account"),
					testCheckTokenExpiresAt("argocd_account_token.expires", int64(expiresInSeconds)),
					resource.TestCheckResourceAttr("argocd_account_token.expires", "expires_in", fmt.Sprintf("%ds", expiresInSeconds)),
					resource.TestCheckResourceAttrSet("argocd_account_token.expires", "id"),
					resource.TestCheckResourceAttrSet("argocd_account_token.expires", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_account_token.expires", "jwt"),
					resource.TestCheckNoResourceAttr("argocd_account_token.expires", "renew_after"),
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
					testCheckTokenExpiresAt("argocd_account_token.expires", int64(expiresInSeconds)),
				),
			},
		},
	})
}

func TestAccArgoCDAccountToken_Multiple(t *testing.T) {
	count := 3 + rand.Intn(7)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "argocd_account_token" "multiple1a" {
	count = %[1]d
}

resource "argocd_account_token" "multiple1b" {
	count = %[1]d
}

resource "argocd_account_token" "multiple2a" {
	account = "test"
	count   = %[1]d
}

resource "argocd_account_token" "multiple2b" {
	account = "test"
	count   = %[1]d
}
				`, count),
				Check: resource.ComposeTestCheckFunc(
					testTokenIssuedAtSet("argocd_account_token.multiple1a", count),
					testTokenIssuedAtSet("argocd_account_token.multiple1b", count),
					testTokenIssuedAtSet("argocd_account_token.multiple2a", count),
					testTokenIssuedAtSet("argocd_account_token.multiple2b", count),
				),
			},
		},
	})
}

func TestAccArgoCDAccountToken_RenewAfter(t *testing.T) {
	renewAfterSeconds := 15
	config := fmt.Sprintf(`
resource "argocd_account_token" "renew_after" {
	renew_after = "%ds"
}
	`, renewAfterSeconds)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("argocd_account_token.renew_after", "account"),
					resource.TestCheckNoResourceAttr("argocd_account_token.renew_after", "expires_at"),
					resource.TestCheckNoResourceAttr("argocd_account_token.renew_after", "expires_in"),
					resource.TestCheckResourceAttrSet("argocd_account_token.renew_after", "id"),
					resource.TestCheckResourceAttrSet("argocd_account_token.renew_after", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_account_token.renew_after", "jwt"),
					resource.TestCheckResourceAttr("argocd_account_token.renew_after", "renew_after", fmt.Sprintf("%ds", renewAfterSeconds)),
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
					resource.TestCheckResourceAttr("argocd_account_token.renew_after", "renew_after", fmt.Sprintf("%ds", renewAfterSeconds)),
				),
			},
		},
	})
}

func TestAccArgoCDAccountToken_UpgradeFromV5(t *testing.T) {
	config := `
resource "argocd_account_token" "upgrade" {
	account     = "test"
	expires_in  = "24h"
	renew_after = "12h"
}`

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: providerV5(),
				Config:            config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("argocd_account_token.upgrade", "account", "test"),
					resource.TestCheckResourceAttrSet("argocd_account_token.upgrade", "expires_at"),
					resource.TestCheckResourceAttr("argocd_account_token.upgrade", "expires_in", "24h"),
					resource.TestCheckResourceAttrSet("argocd_account_token.upgrade", "id"),
					resource.TestCheckResourceAttrSet("argocd_account_token.upgrade", "issued_at"),
					resource.TestCheckResourceAttrSet("argocd_account_token.upgrade", "jwt"),
					resource.TestCheckResourceAttr("argocd_account_token.upgrade", "renew_after", "12h"),
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
