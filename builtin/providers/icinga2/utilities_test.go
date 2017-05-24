package icinga2

import (
	"fmt"

	"github.com/r3labs/terraform/helper/resource"
	"github.com/r3labs/terraform/terraform"
)

func testAccCheckResourceState(resourceName, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
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
