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
					/* testAccCheckBundleExists("circonus_check.usage", "foo"), */
					resource.TestCheckResourceAttr("circonus_check.usage", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.2388330941.id", "/broker/1"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.http_headers.%", "3"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.http_headers.Accept", "application/json"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.http_headers.X-Circonus-App-Name", "TerraformCheck"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.http_headers.X-Circonus-Auth-Token", "<env 'CIRCONUS_API_TOKEN'>"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.http_version", "1.0"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.method", "GET"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.port", "443"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.read_limit", "1048576"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.redirects", "3"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "config.0.url", "https://api.circonus.com/account/current"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.711946474.name", "_usage`0`_used"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.711946474.tags.%", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.711946474.tags.source", "circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.711946474.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.711946474.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.2238926521.name", "_usage`0`_limit"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.2238926521.tags.%", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.2238926521.tags.source", "circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.2238926521.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.2238926521.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "period", "300s"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.%", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.source", "circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "target", "api.circonus.com"),
					resource.TestCheckResourceAttr("circonus_check.usage", "type", "json"),
				),
			},
		},
	})
}

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

const testAccCirconusCheckBundleConfig = `
variable "usage_default_unit" {
  default = "qty"
}

resource "circonus_metric" "limit" {
  name = "_usage` + "`0`" + `_limit"

  tags = {
    source = "circonus"
  }

  type = "numeric"
  unit = "${var.usage_default_unit}"
}

resource "circonus_metric" "used" {
  name = "_usage` + "`0`" + `_used"

  tags = {
    source = "circonus"
  }

  type = "numeric"
  unit = "${var.usage_default_unit}"
}

resource "circonus_check" "usage" {
  active = true
  name = "Terraform test: api.circonus.com metric usage check"
  notes = "notes!"
  period = "300s"

  collector {
    id = "/broker/1"
  }

  json {
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

//  steams = "${circonus_metric.*.stream}"

  stream {
    name = "${circonus_metric.used.name}"
    tags = "${circonus_metric.used.tags}"
    type = "${circonus_metric.used.type}"
    unit = "${coalesce(circonus_metric.used.unit, var.usage_default_unit)}"
  }

  stream {
    name = "${circonus_metric.limit.name}"
    tags = "${circonus_metric.limit.tags}"
    type = "${circonus_metric.limit.type}"
    unit = "${coalesce(circonus_metric.limit.unit, var.usage_default_unit)}"
  }

  tags = {
    source = "circonus",
  }
}
`
