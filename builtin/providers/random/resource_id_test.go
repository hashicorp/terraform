package random

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

type idLens struct {
	b64Len    int
	b64UrlLen int
	b64StdLen int
	hexLen    int
}

func TestAccResourceID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceIDConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceIDCheck("random_id.foo", &idLens{
						b64Len:    6,
						b64UrlLen: 6,
						b64StdLen: 8,
						hexLen:    8,
					}),
					testAccResourceIDCheck("random_id.bar", &idLens{
						b64Len:    12,
						b64UrlLen: 12,
						b64StdLen: 14,
						hexLen:    14,
					}),
				),
			},
		},
	})
}

func testAccResourceIDCheck(id string, want *idLens) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("Not found: %s", id)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		b64Str := rs.Primary.Attributes["b64"]
		b64UrlStr := rs.Primary.Attributes["b64_url"]
		b64StdStr := rs.Primary.Attributes["b64_std"]
		hexStr := rs.Primary.Attributes["hex"]
		decStr := rs.Primary.Attributes["dec"]

		if got, want := len(b64Str), want.b64Len; got != want {
			return fmt.Errorf("base64 string length is %d; want %d", got, want)
		}
		if got, want := len(b64UrlStr), want.b64UrlLen; got != want {
			return fmt.Errorf("base64 URL string length is %d; want %d", got, want)
		}
		if got, want := len(b64StdStr), want.b64StdLen; got != want {
			return fmt.Errorf("base64 STD string length is %d; want %d", got, want)
		}
		if got, want := len(hexStr), want.hexLen; got != want {
			return fmt.Errorf("hex string length is %d; want %d", got, want)
		}
		if len(decStr) < 1 {
			return fmt.Errorf("decimal string is empty; want at least one digit")
		}

		return nil
	}
}

const (
	testAccResourceIDConfig = `
resource "random_id" "foo" {
  byte_length = 4
}

resource "random_id" "bar" {
  byte_length = 4
	prefix      = "cloud-"
}
`
)
