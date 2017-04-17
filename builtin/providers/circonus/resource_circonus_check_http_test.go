package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckHTTP_basic(t *testing.T) {
	checkName := fmt.Sprintf("Terraform test: noit's jezebel availability check - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckHTTPConfigFmt, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.jezebel", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.auth_method", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.auth_password", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.auth_user", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.body_regexp", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.ciphers", ""),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.code", `^200$`),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.extract", `HTTP/1.1 200 OK`),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.key_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.payload", ""),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.headers.%", "1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.headers.Host", "127.0.0.1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.version", "1.1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.method", "GET"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.read_limit", "1048576"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.4213422905.url", "http://127.0.0.1:8083/resmon"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "notes", "Check to make sure jezebel is working as expected"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.#", "4"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.name", "code"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2380257438.type", "text"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.name", "duration"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.3634949264.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.name", "tt_connect"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.1717167158.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.name", "tt_firstbyte"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "metric.2305894402.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "target", "127.0.0.1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "type", "http"),
				),
			},
		},
	})
}

const testAccCirconusCheckHTTPConfigFmt = `
variable "http_check_tags" {
  type = "list"
  default = [ "app:circonus", "app:jezebel", "lifecycle:unittest", "source:circonus" ]
}

resource "circonus_metric" "status_code" {
  name = "code"
  tags = [ "${var.http_check_tags}" ]
  type = "text"
}

resource "circonus_metric" "request_duration" {
  name = "duration"
  tags = [ "${var.http_check_tags}" ]
  type = "numeric"
  unit = "seconds"
}

resource "circonus_metric" "request_ttconnect" {
  name = "tt_connect"
  tags = [ "${var.http_check_tags}" ]
  type = "numeric"
  unit = "milliseconds"
}

resource "circonus_metric" "request_ttfb" {
  name = "tt_firstbyte"
  tags = [ "${var.http_check_tags}" ]
  type = "numeric"
  unit = "milliseconds"
}

resource "circonus_check" "jezebel" {
  active = true
  name = "%s"
  notes = "Check to make sure jezebel is working as expected"
  period = "60s"

  collector {
    id = "/broker/1"
  }

  http {
    code = "^200$"
    extract     = "HTTP/1.1 200 OK"
    headers     = {
      Host = "127.0.0.1",
    }
    version     = "1.1"
    method      = "GET"
    read_limit  = 1048576
    url         = "http://127.0.0.1:8083/resmon"
  }

  metric {
    name = "${circonus_metric.status_code.name}"
    tags = [ "${circonus_metric.status_code.tags}" ]
    type = "${circonus_metric.status_code.type}"
  }

  metric {
    name = "${circonus_metric.request_duration.name}"
    tags = [ "${circonus_metric.request_duration.tags}" ]
    type = "${circonus_metric.request_duration.type}"
    unit = "${circonus_metric.request_duration.unit}"
  }

  metric {
    name = "${circonus_metric.request_ttconnect.name}"
    tags = [ "${circonus_metric.request_ttconnect.tags}" ]
    type = "${circonus_metric.request_ttconnect.type}"
    unit = "${circonus_metric.request_ttconnect.unit}"
  }

  metric {
    name = "${circonus_metric.request_ttfb.name}"
    tags = [ "${circonus_metric.request_ttfb.tags}" ]
    type = "${circonus_metric.request_ttfb.type}"
    unit = "${circonus_metric.request_ttfb.unit}"
  }

  tags = [ "${var.http_check_tags}" ]
}
`
