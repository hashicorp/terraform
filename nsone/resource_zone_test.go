package nsone

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccZone_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccZone_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneState("zone", "example.com"),
				),
			},
		},
	})
}

func testAccCheckZoneState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_zone.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "nsone_zone.foobar")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := rs.Primary
		if p.Attributes[key] != value {
			return fmt.Errorf(
				"%s != %s (actual: %s)", key, value, p.Attributes[key])
		}

		return nil
	}
}

const testAccZone_basic = `
resource "nsone_zone" "foobar" {
	zone = "example.com"
	hostmaster = "example.com"
}`
