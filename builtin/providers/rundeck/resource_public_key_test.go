package rundeck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/apparentlymart/go-rundeck-api/rundeck"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPublicKey_basic(t *testing.T) {
	var key rundeck.KeyMeta
	var keyMaterial string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPublicKeyCheckDestroy(&key),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPublicKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPublicKeyCheckExists("rundeck_public_key.test", &key, &keyMaterial),
					func(s *terraform.State) error {
						if expected := "keys/terraform_acceptance_tests/public_key"; key.Path != expected {
							return fmt.Errorf("wrong path; expected %v, got %v", expected, key.Path)
						}
						if !strings.HasSuffix(key.URL, "/storage/keys/terraform_acceptance_tests/public_key") {
							return fmt.Errorf("wrong URL; expected to end with the key path")
						}
						if expected := "file"; key.ResourceType != expected {
							return fmt.Errorf("wrong resource type; expected %v, got %v", expected, key.ResourceType)
						}
						if expected := "public"; key.KeyType != expected {
							return fmt.Errorf("wrong key type; expected %v, got %v", expected, key.KeyType)
						}
						if !strings.Contains(keyMaterial, "test+public+key+for+terraform") {
							return fmt.Errorf("wrong key material")
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccPublicKeyCheckDestroy(key *rundeck.KeyMeta) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*rundeck.Client)
		_, err := client.GetKeyMeta(key.Path)
		if err == nil {
			return fmt.Errorf("key still exists")
		}
		if _, ok := err.(*rundeck.NotFoundError); !ok {
			return fmt.Errorf("got something other than NotFoundError (%v) when getting key", err)
		}

		return nil
	}
}

func testAccPublicKeyCheckExists(rn string, key *rundeck.KeyMeta, keyMaterial *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("key id not set")
		}

		client := testAccProvider.Meta().(*rundeck.Client)
		gotKey, err := client.GetKeyMeta(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting key metadata: %s", err)
		}

		*key = *gotKey

		*keyMaterial, err = client.GetKeyContent(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting key contents: %s", err)
		}

		return nil
	}
}

const testAccPublicKeyConfig_basic = `
resource "rundeck_public_key" "test" {
  path = "terraform_acceptance_tests/public_key"
  key_material = "ssh-rsa test+public+key+for+terraform nobody@nowhere"
}
`
