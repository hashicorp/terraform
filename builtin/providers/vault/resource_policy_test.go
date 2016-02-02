package vault

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestAccVaultPolicy_basic(t *testing.T) {
	name := fmt.Sprintf("policy-%s", acctest.RandString(10))
	rules := `path "sys/*" {
		policy = "deny"
	}`
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultPolicyConfig(name, rules),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultPolicyExists("vault_policy.foo"),
					testAccCheckVaultPolicyAttributes(name, rules),
				),
			},
		},
	})
}

func TestAccVaultPolicy_disappears(t *testing.T) {
	name := fmt.Sprintf("policy-%s", acctest.RandString(10))
	rules := `path "sys/*" {
		policy = "deny"
	}`
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultPolicyConfig(name, rules),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultPolicyExists("vault_policy.foo"),
					testAccVaultPolicyDisappear(name),
				),
				ExpectNonEmptyPlan: true,
			},
			// Follow up w/ empty config should be empty, since the policy is gone.
			resource.TestStep{
				Config: "",
			},
		},
	})
}

func TestAccVaultPolicy_ruleDrift(t *testing.T) {
	name := fmt.Sprintf("policy-%s", acctest.RandString(10))
	rules := `path "sys/*" {
		policy = "deny"
	}`
	driftedRules := `path "sys/*" {
		policy = "allow"
	}`
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultPolicyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultPolicyConfig(name, rules),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultPolicyExists("vault_policy.foo"),
					testAccVaultPolicyDrift(name, driftedRules),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckVaultPolicyExists(
	key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[key]
		client := testAccProvider.Meta().(*api.Client)

		policies, err := client.Sys().ListPolicies()
		if err != nil {
			return fmt.Errorf("Error listing policies: %s", err)
		}

		for _, policy := range policies {
			if policy == rs.Primary.ID {
				return nil
			}
		}
		return fmt.Errorf("Policy not found: %s", rs.Primary.ID)
	}
}

func testAccCheckVaultPolicyAttributes(
	name string,
	expectedRules string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)

		rules, err := client.Sys().GetPolicy(name)
		if err != nil {
			return fmt.Errorf("Error getting policy: %s", err)
		}
		if rules != expectedRules {
			return fmt.Errorf("Expected policy rules %q, got %q",
				expectedRules, rules)
		}
		return nil
	}
}

func testAccCheckVaultPolicyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*api.Client)

	policies, err := client.Sys().ListPolicies()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_policy" {
			continue
		}
		for _, policyName := range policies {
			if policyName == rs.Primary.ID {
				return fmt.Errorf("Policy still exists: %s", policyName)
			}
		}
	}

	return nil
}

func testAccVaultPolicyDisappear(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		return client.Sys().DeletePolicy(name)
	}
}

func testAccVaultPolicyDrift(name, rules string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*api.Client)
		return client.Sys().PutPolicy(name, rules)
	}
}

func testAccVaultPolicyConfig(
	name, rules string) string {
	return fmt.Sprintf(`
resource "vault_policy" "foo" {
  name = "%s"
  rules = %q
}
`, name, rules)
}
