package random

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccResourceIDConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceIDCheck("random_id.foo"),
				),
			},
		},
	})
}

func testAccResourceIDCheck(id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		b64Str := rs.Primary.Attributes["b64"]
		hexStr := rs.Primary.Attributes["hex"]
		decStr := rs.Primary.Attributes["dec"]

		if got, want := len(b64Str), 6; got != want {
			return fmt.Errorf("base64 string length is %d; want %d", got, want)
		}
		if got, want := len(hexStr), 8; got != want {
			return fmt.Errorf("hex string length is %d; want %d", got, want)
		}
		if len(decStr) < 1 {
			return fmt.Errorf("decimal string is empty; want at least one digit")
		}

		return nil
	}
}

const testAccResourceIDConfig = `
resource "random_id" "foo" {
    byte_length = 4
}
`
