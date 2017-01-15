package circonus

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckJSON_basic(t *testing.T) {
	const jsonHash = "2883347764"
	jsonAttr := func(key string) string {
		keyParts := []string{string(_CheckJSONAttr), jsonHash, key}
		return strings.Join(keyParts, ".")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusCheckJSONConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.usage", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("auth_method"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("auth_password"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("auth_user"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("ca_chain"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("certificate_file"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("ciphers"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("key_file"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("payload"), ""),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("headers.%"), "3"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("headers.Accept"), "application/json"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("headers.X-Circonus-App-Name"), "TerraformCheck"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("headers.X-Circonus-Auth-Token"), "<env 'CIRCONUS_API_TOKEN'>"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("version"), "1.0"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("method"), "GET"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("port"), "443"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("read_limit"), "1048576"),
					resource.TestCheckResourceAttr("circonus_check.usage", jsonAttr("url"), "https://api.circonus.com/account/current"),
					resource.TestCheckResourceAttr("circonus_check.usage", "name", "Terraform test: api.circonus.com metric usage check"),
					resource.TestCheckResourceAttr("circonus_check.usage", "notes", "notes!"),
					resource.TestCheckResourceAttr("circonus_check.usage", "period", "300s"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.#", "2"),

					resource.TestCheckResourceAttr("circonus_check.usage", "stream.1992097900.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.1992097900.name", "_usage`0`_limit"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.1992097900.tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.1992097900.tags.1384943139", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.1992097900.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.1992097900.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.3280673139.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.3280673139.name", "_usage`0`_used"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.3280673139.tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.3280673139.tags.1384943139", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.3280673139.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "stream.3280673139.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.1384943139", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.3579657361", "source:unittest"),
					resource.TestCheckResourceAttr("circonus_check.usage", "target", "api.circonus.com"),
					resource.TestCheckResourceAttr("circonus_check.usage", "type", "json"),
				),
			},
		},
	})
}

const testAccCirconusCheckJSONConfig = `
variable "usage_default_unit" {
  default = "qty"
}

resource "circonus_metric" "limit" {
  name = "_usage` + "`0`" + `_limit"

  tags = [ "source:circonus" ]

  type = "numeric"
  unit = "${var.usage_default_unit}"
}

resource "circonus_metric" "used" {
  name = "_usage` + "`0`" + `_used"

  tags = [ "source:circonus" ]

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
    headers = {
      Accept                = "application/json",
      X-Circonus-App-Name   = "TerraformCheck",
      X-Circonus-Auth-Token = "<env 'CIRCONUS_API_TOKEN'>",
    }
    version = "1.0"
    method = "GET"
    port = 443
    read_limit = 1048576
  }

  stream {
    name = "${circonus_metric.used.name}"
    tags = [ "${circonus_metric.used.tags}" ]
    type = "${circonus_metric.used.type}"
    unit = "${coalesce(circonus_metric.used.unit, var.usage_default_unit)}"
  }

  stream {
    name = "${circonus_metric.limit.name}"
    tags = [ "${circonus_metric.limit.tags}" ]
    type = "${circonus_metric.limit.type}"
    unit = "${coalesce(circonus_metric.limit.unit, var.usage_default_unit)}"
  }

  tags = [ "source:circonus", "source:unittest" ]
}
`
