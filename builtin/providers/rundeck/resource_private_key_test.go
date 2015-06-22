package rundeck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/apparentlymart/go-rundeck-api/rundeck"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPrivateKey_basic(t *testing.T) {
	var key rundeck.KeyMeta

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPrivateKeyCheckDestroy(&key),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPrivateKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPrivateKeyCheckExists("rundeck_private_key.test", &key),
					func(s *terraform.State) error {
						if expected := "keys/terraform_acceptance_tests/private_key"; key.Path != expected {
							return fmt.Errorf("wrong path; expected %v, got %v", expected, key.Path)
						}
						if !strings.HasSuffix(key.URL, "/storage/keys/terraform_acceptance_tests/private_key") {
							return fmt.Errorf("wrong URL; expected to end with the key path")
						}
						if expected := "file"; key.ResourceType != expected {
							return fmt.Errorf("wrong resource type; expected %v, got %v", expected, key.ResourceType)
						}
						if expected := "private"; key.KeyType != expected {
							return fmt.Errorf("wrong key type; expected %v, got %v", expected, key.KeyType)
						}
						// Rundeck won't let us re-retrieve a private key payload, so we can't test
						// that the key material was submitted and stored correctly.
						return nil
					},
				),
			},
		},
	})
}

func testAccPrivateKeyCheckDestroy(key *rundeck.KeyMeta) resource.TestCheckFunc {
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

func testAccPrivateKeyCheckExists(rn string, key *rundeck.KeyMeta) resource.TestCheckFunc {
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

		return nil
	}
}

const testAccPrivateKeyConfig_basic = `
resource "rundeck_private_key" "test" {
  path = "terraform_acceptance_tests/private_key"
  key_material = "this is not a real private key"
}
`
