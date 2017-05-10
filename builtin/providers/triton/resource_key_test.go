package triton

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/triton-go"
)

func TestAccTritonKey_basic(t *testing.T) {
	keyName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	publicKeyMaterial, _, err := acctest.RandSSHKeyPair("TestAccTritonKey_basic@terraform")
	if err != nil {
		t.Fatalf("Cannot generate test SSH key pair: %s", err)
	}
	config := testAccTritonKey_basic(keyName, publicKeyMaterial)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonKeyExists("triton_key.test"),
					resource.TestCheckResourceAttr("triton_key.test", "name", keyName),
					resource.TestCheckResourceAttr("triton_key.test", "key", publicKeyMaterial),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
		},
	})
}

func TestAccTritonKey_noKeyName(t *testing.T) {
	keyComment := fmt.Sprintf("acctest_%d@terraform", acctest.RandInt())
	keyMaterial, _, err := acctest.RandSSHKeyPair(keyComment)
	if err != nil {
		t.Fatalf("Cannot generate test SSH key pair: %s", err)
	}
	config := testAccTritonKey_noKeyName(keyMaterial)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonKeyExists("triton_key.test"),
					resource.TestCheckResourceAttr("triton_key.test", "name", keyComment),
					resource.TestCheckResourceAttr("triton_key.test", "key", keyMaterial),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
		},
	})
}

func testCheckTritonKeyExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*triton.Client)

		key, err := conn.Keys().GetKey(context.Background(), &triton.GetKeyInput{
			KeyName: rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("Bad: Check Key Exists: %s", err)
		}

		if key == nil {
			return fmt.Errorf("Bad: Key %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*triton.Client)

	return resource.Retry(1*time.Minute, func() *resource.RetryError {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "triton_key" {
				continue
			}

			key, err := conn.Keys().GetKey(context.Background(), &triton.GetKeyInput{
				KeyName: rs.Primary.ID,
			})
			if err != nil {
				return nil
			}

			if key != nil {
				return resource.RetryableError(fmt.Errorf("Bad: Key %q still exists", rs.Primary.ID))
			}
		}

		return nil
	})
}

var testAccTritonKey_basic = func(keyName string, keyMaterial string) string {
	return fmt.Sprintf(`resource "triton_key" "test" {
	    name = "%s"
	    key = "%s"
	}
	`, keyName, keyMaterial)
}

var testAccTritonKey_noKeyName = func(keyMaterial string) string {
	return fmt.Sprintf(`resource "triton_key" "test" {
	    key = "%s"
	}
	`, keyMaterial)
}
