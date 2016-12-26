package circonus

import (
	"fmt"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusCheckBundle_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusCheckBundleConfig,
				Check: resource.ComposeTestCheckFunc(
					/* testAccCheckBundleExists("circonus_check_bundle.usage_check", "foo"), */
					resource.TestCheckResourceAttr("circonus_check_bundle.usage_check", "name", "Terraform test: api.circonus.com metric usage check"),
					resource.TestCheckResourceAttr("circonus_check_bundle.usage_check", "target", "api.circonus.com"),
					resource.TestCheckResourceAttr("circonus_check_bundle.usage_check", "type", "http"),
				),
			},
		},
	})
}

func testAccCheckDestroyCirconusCheckBundle(s *terraform.State) error {
	c := testAccProvider.Meta().(*api.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_check_bundle" {
			continue
		}

		exists, err := checkCheckBundleExists(c, api.CIDType(rs.Primary.ID))
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

		client := testAccProvider.Meta().(*api.API)
		exists, err := checkCheckBundleExists(client, api.CIDType(rs.Primary.ID))

		if err != nil {
			return fmt.Errorf("Error checking check_bundle %s", err)
		}

		if !exists {
			return fmt.Errorf("check_bundle not found")
		}

		return nil
	}
}

func checkCheckBundleExists(c *api.API, checkBundleID api.CIDType) (bool, error) {
	cb, err := c.FetchCheckBundleByCID(checkBundleID)
	if err != nil {
		return false, err
	}

	if api.CIDType(cb.CID) == checkBundleID {
		return true, nil
	} else {
		return false, nil
	}
}

const testAccCirconusCheckBundleConfig = `
resource "circonus_check_bundle" "usage_check" {
  name = "Terraform test: api.circonus.com metric usage check"
  type = "http"
  target = "api.circonus.com"
  brokers = [
    "/broker/1",
  ]
  metrics {
    name = "_usage` + "`0`" + `_used"
    tags = ["source:circonus"]
    type = "numeric"
  }
  config {
    url = "https://api.circonus.com/account/current"
  }
  tags = ["source:circonus","creator:terraform"]
}
`
