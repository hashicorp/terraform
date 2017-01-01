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
					/* testAccCheckBundleExists("circonus_check.usage_check", "foo"), */
					resource.TestCheckResourceAttr("circonus_check.usage_check", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "brokers.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "brokers.0", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.http_headers.%", "3"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.http_headers.Accept", "application/json"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.http_headers.X-Circonus-App-Name", "TerraformCheck"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.http_headers.X-Circonus-Auth-Token", "<env 'CIRCONUS_API_TOKEN'>"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.http_version", "1.0"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.method", "GET"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.port", "443"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.read_limit", "1048576"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.redirects", "3"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "config.0.url", "https://api.circonus.com/account/current"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "metric.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "metric.0.name", "_usage`0`_used"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "metric.0.tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "metric.0.tags.0", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "metric.0.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "name", "Terraform test: api.circonus.com metric usage check"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "period", "60"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "tags.0", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "tags.1", "creator:terraform"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "target", "api.circonus.com"),
					resource.TestCheckResourceAttr("circonus_check.usage_check", "type", "json"),
				),
			},
		},
	})
}

func testAccCheckDestroyCirconusCheckBundle(s *terraform.State) error {
	c := testAccProvider.Meta().(*api.API)

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

		client := testAccProvider.Meta().(*api.API)
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

func checkCheckBundleExists(c *api.API, checkBundleID api.CIDType) (bool, error) {
	cb, err := c.FetchCheckBundle(checkBundleID)
	if err != nil {
		return false, err
	}

	if api.CIDType(&cb.CID) == checkBundleID {
		return true, nil
	} else {
		return false, nil
	}
}

const testAccCirconusCheckBundleConfig = `
resource "circonus_check" "usage_check" {
  active = true
  name = "Terraform test: api.circonus.com metric usage check"
  type = "json"
  target = "api.circonus.com"
  period = 60
  brokers = [
    "/broker/1",
  ]
  metric {
    name = "_usage` + "`0`" + `_used"
    tags = ["source:circonus"]
    type = "numeric"
  }
  config {
    url = "https://api.circonus.com/account/current"

    http_headers = {
      "Accept" = "application/json",
      "X-Circonus-App-Name" = "TerraformCheck",
      "X-Circonus-Auth-Token" = "<env 'CIRCONUS_API_TOKEN'>",
    }
    http_version = "1.0"
    method = "GET"
    port = 443
    read_limit = 1048576
    redirects = 3
  }
  tags = ["source:circonus","creator:terraform"]
}
`
