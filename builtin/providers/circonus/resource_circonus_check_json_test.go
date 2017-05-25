package circonus

import (
	"regexp"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckJSON_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusCheckJSONConfig1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.usage", "active", "true"),
					resource.TestMatchResourceAttr("circonus_check.usage", "check_id", regexp.MustCompile(config.CheckCIDRegex)),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.auth_method", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.auth_password", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.auth_user", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.ciphers", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.key_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.payload", ""),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.headers.%", "3"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.headers.Accept", "application/json"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.headers.X-Circonus-App-Name", "TerraformCheck"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.headers.X-Circonus-Auth-Token", "<env 'CIRCONUS_API_TOKEN'>"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.version", "1.0"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.method", "GET"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.port", "443"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.read_limit", "1048576"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.2626248092.url", "https://api.circonus.com/account/current"),
					resource.TestCheckResourceAttr("circonus_check.usage", "name", "Terraform test: api.circonus.com metric usage check"),
					resource.TestCheckResourceAttr("circonus_check.usage", "notes", ""),
					resource.TestCheckResourceAttr("circonus_check.usage", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.name", "_usage`0`_limit"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.name", "_usage`0`_used"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.usage", "target", "api.circonus.com"),
					resource.TestCheckResourceAttr("circonus_check.usage", "type", "json"),
				),
			},
			{
				Config: testAccCirconusCheckJSONConfig2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.usage", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.auth_method", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.auth_password", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.auth_user", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.ciphers", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.key_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.payload", ""),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.headers.%", "3"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.headers.Accept", "application/json"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.headers.X-Circonus-App-Name", "TerraformCheck"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.headers.X-Circonus-Auth-Token", "<env 'CIRCONUS_API_TOKEN'>"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.version", "1.1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.method", "GET"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.port", "443"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.read_limit", "1048576"),
					resource.TestCheckResourceAttr("circonus_check.usage", "json.3951979786.url", "https://api.circonus.com/account/current"),
					resource.TestCheckResourceAttr("circonus_check.usage", "name", "Terraform test: api.circonus.com metric usage check"),
					resource.TestCheckResourceAttr("circonus_check.usage", "notes", "notes!"),
					resource.TestCheckResourceAttr("circonus_check.usage", "period", "300s"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.name", "_usage`0`_limit"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.1992097900.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.name", "_usage`0`_used"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.usage", "metric.3280673139.unit", "qty"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.usage", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.usage", "target", "api.circonus.com"),
					resource.TestCheckResourceAttr("circonus_check.usage", "type", "json"),
				),
			},
		},
	})
}

const testAccCirconusCheckJSONConfig1 = `
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
  period = "60s"

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

  metric {
    name = "${circonus_metric.used.name}"
    tags = [ "${circonus_metric.used.tags}" ]
    type = "${circonus_metric.used.type}"
    unit = "${coalesce(circonus_metric.used.unit, var.usage_default_unit)}"
  }

  metric {
    name = "${circonus_metric.limit.name}"
    tags = [ "${circonus_metric.limit.tags}" ]
    type = "${circonus_metric.limit.type}"
    unit = "${coalesce(circonus_metric.limit.unit, var.usage_default_unit)}"
  }

  tags = [ "source:circonus", "lifecycle:unittest" ]
}
`

const testAccCirconusCheckJSONConfig2 = `
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
    version = "1.1"
    method = "GET"
    port = 443
    read_limit = 1048576
  }

  metric {
    name = "${circonus_metric.used.name}"
    tags = [ "${circonus_metric.used.tags}" ]
    type = "${circonus_metric.used.type}"
    unit = "${coalesce(circonus_metric.used.unit, var.usage_default_unit)}"
  }

  metric {
    name = "${circonus_metric.limit.name}"
    tags = [ "${circonus_metric.limit.tags}" ]
    type = "${circonus_metric.limit.type}"
    unit = "${coalesce(circonus_metric.limit.unit, var.usage_default_unit)}"
  }

  tags = [ "source:circonus", "lifecycle:unittest" ]
}
`
