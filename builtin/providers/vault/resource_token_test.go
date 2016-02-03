package vault

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestAccVaultToken_basic(t *testing.T) {
	var token api.Secret
	displayName := acctest.RandString(10)
	// Vault prefixes token display names with token-
	expectedDisplayName := fmt.Sprintf("token-%s", displayName)

	meta := map[string]string{"foo": "bar"}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultTokenDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultTokenConfig(
					displayName, "1m", 5, []string{"policy"}, true, meta),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultTokenExists("vault_token.foo", &token),
					testAccCheckVaultTokenAttributes(&token,
						expectedDisplayName, "1m", 5, []string{"policy"}, true, meta),
				),
			},
		},
	})
}

func TestAccVaultToken_disappears(t *testing.T) {
	var token api.Secret
	displayName := acctest.RandString(10)

	meta := map[string]string{"foo": "bar"}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultTokenDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultTokenConfig(
					displayName, "1m", 5, []string{"policy"}, true, meta),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultTokenExists("vault_token.foo", &token),
					testAccVaultTokenDisappear(&token),
				),
				ExpectNonEmptyPlan: true,
			},
			// Empty config should yield empty plan, since token is gone
			resource.TestStep{
				Config: "",
			},
		},
	})
}

func TestAccVaultToken_implicitParams(t *testing.T) {
	var token api.Secret
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVaultTokenDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVaultTokenConfigMinimal(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultTokenExists("vault_token.foo", &token),
					resource.TestCheckResourceAttr(
						"vault_token.foo", "no_default_policy", "false"),
					resource.TestCheckResourceAttr(
						"vault_token.foo", "policies.#", "1"),
					resource.TestCheckResourceAttr(
						"vault_token.foo", "policies.2678487596", "root"),
				),
			},
		},
	})
}

func testAccCheckVaultTokenExists(key string, token *api.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[key]
		client := testAccProvider.Meta().(*api.Client)

		t, err := client.Auth().Token().Lookup(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error looking up token: %s", err)
		}

		*token = *t
		return nil
	}
}

func testAccCheckVaultTokenAttributes(
	token *api.Secret,
	expectedDisplayName string,
	expectedTTL string,
	expectedNumUses int,
	expectedPolicyNames []string,
	expectedNoDefaultPolicy bool,
	expectedMeta map[string]string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		gotDisplayName := token.Data["display_name"].(string)
		if gotDisplayName != expectedDisplayName {
			return fmt.Errorf("Expected display name %q, got %q",
				expectedDisplayName, gotDisplayName)
		}

		expectedDuration, _ := time.ParseDuration(expectedTTL)
		gotDuration := time.Duration(token.Data["ttl"].(float64)) * time.Second
		if gotDuration != expectedDuration {
			return fmt.Errorf("Expected TTL %s, got %d",
				expectedDuration, gotDuration)
		}

		gotNumUses := int(token.Data["num_uses"].(float64))
		if gotNumUses != expectedNumUses {
			return fmt.Errorf("Expected num uses %d, got %d",
				expectedNumUses, gotNumUses)
		}

		gotDefaultPolicy := false
		gotPolicyNames := []string{}
		for _, policyName := range token.Data["policies"].([]interface{}) {
			if policyName.(string) == "default" {
				gotDefaultPolicy = true
			} else {
				gotPolicyNames = append(gotPolicyNames, policyName.(string))
			}
		}

		if !reflect.DeepEqual(gotPolicyNames, expectedPolicyNames) {
			return fmt.Errorf("Expected policies %v, got %v",
				expectedPolicyNames, gotPolicyNames)
		}

		if gotDefaultPolicy && expectedNoDefaultPolicy {
			return fmt.Errorf("Expected no default profile, but got one!")
		}

		gotMeta := make(map[string]string)
		for k, v := range token.Data["meta"].(map[string]interface{}) {
			gotMeta[k] = v.(string)
		}
		if !reflect.DeepEqual(gotMeta, expectedMeta) {
			return fmt.Errorf("Expected meta %v, got %v", expectedMeta, gotMeta)
		}

		return nil
	}
}

func testAccCheckVaultTokenDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*api.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vault_token" {
			continue
		}

		_, err := client.Auth().Token().Lookup(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Token still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccVaultTokenDisappear(token *api.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id := token.Data["id"].(string)
		client := testAccProvider.Meta().(*api.Client)
		return client.Auth().Token().RevokeTree(id)
	}
}

func testAccVaultTokenConfig(
	displayName, ttl string,
	numUses int,
	policyNames []string,
	noDefaultPolicy bool,
	meta map[string]string) string {
	var m bytes.Buffer
	for k, v := range meta {
		m.WriteString(fmt.Sprintf("    %s = %q\n", k, v))
	}
	policies := []string{}
	for _, p := range policyNames {
		policies = append(policies, fmt.Sprintf("%q", p))
	}
	return fmt.Sprintf(`
resource "vault_token" "foo" {
	display_name       = "%s"
	ttl                = "%s"
	num_uses           = %d
	policies           = [%s]
	no_default_policy = %t
	meta {
		%s
	}
}
`, displayName, ttl, numUses, strings.Join(policies, ", "),
		noDefaultPolicy, m.String())
}

func testAccVaultTokenConfigMinimal() string {
	return `
resource "vault_token" "foo" {
}
`
}
