package circonus

import (
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func testAccCheckDestroyCirconusCheckBundle(s *terraform.State) error {
	c := testAccProvider.Meta().(*_ProviderContext)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_check" {
			continue
		}

		cid := rs.Primary.ID
		exists, err := checkCheckBundleExists(c, api.CIDType(&cid))
		if err != nil {
			return fmt.Errorf("Error checking check bundle %s", err)
		}

		if exists {
			return fmt.Errorf("check bundle still exists after destroy")
		}
	}

	return nil
}

func testAccCheckBundleExists(n string, checkBundleID api.CIDType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*_ProviderContext)
		cid := rs.Primary.ID
		exists, err := checkCheckBundleExists(client, api.CIDType(&cid))

		if err != nil {
			return fmt.Errorf("Error checking check %s", err)
		}

		if !exists {
			return fmt.Errorf("check not found")
		}

		return nil
	}
}

func checkCheckBundleExists(c *_ProviderContext, checkBundleID api.CIDType) (bool, error) {
	cb, err := c.client.FetchCheckBundle(checkBundleID)
	if err != nil {
		return false, err
	}

	if api.CIDType(&cb.CID) == checkBundleID {
		return true, nil
	} else {
		return false, nil
	}
}
