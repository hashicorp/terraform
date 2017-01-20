package circonus

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckHTTP_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusCheckHTTPConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.jezebel", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "collector.1893401625.id", "/broker/1286"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.auth_method", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.auth_password", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.auth_user", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.body_regexp", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.ciphers", ""),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.code", `^200$`),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.extract", `HTTP/1.1 200 OK`),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.key_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.payload", ""),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.headers.%", "1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.headers.Host", "127.0.0.1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.version", "1.1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.method", "GET"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.read_limit", "1048576"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "http.2201265081.url", "http://127.0.0.1:8083/resmon"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "name", "Terraform test: noit's jezebel availability check"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "notes", "Check to make sure jezebel is working as expected"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.#", "4"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.name", "code"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.42262635.type", "text"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.name", "duration"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.1136493216.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.name", "tt_connect"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.4246441943.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.name", "tt_firstbyte"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "stream.3695203246.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.3219687752", "app:jezebel"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "target", "127.0.0.1"),
					resource.TestCheckResourceAttr("circonus_check.jezebel", "type", "http"),
				),
			},
		},
	})
}

const testAccCirconusCheckHTTPConfig = `
variable "http_check_tags" {
  type = "list"
  default = [ "app:circonus", "app:jezebel", "lifecycle:unittests", "source:circonus" ]
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
  name = "Terraform test: noit's jezebel availability check"
  notes = "Check to make sure jezebel is working as expected"
  period = "60s"

  collector {
    id = "/broker/1286"
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

  stream {
    name = "${circonus_metric.status_code.name}"
    tags = [ "${circonus_metric.status_code.tags}" ]
    type = "${circonus_metric.status_code.type}"
  }

  stream {
    name = "${circonus_metric.request_duration.name}"
    tags = [ "${circonus_metric.request_duration.tags}" ]
    type = "${circonus_metric.request_duration.type}"
    unit = "${circonus_metric.request_duration.unit}"
  }

  stream {
    name = "${circonus_metric.request_ttconnect.name}"
    tags = [ "${circonus_metric.request_ttconnect.tags}" ]
    type = "${circonus_metric.request_ttconnect.type}"
    unit = "${circonus_metric.request_ttconnect.unit}"
  }

  stream {
    name = "${circonus_metric.request_ttfb.name}"
    tags = [ "${circonus_metric.request_ttfb.tags}" ]
    type = "${circonus_metric.request_ttfb.type}"
    unit = "${circonus_metric.request_ttfb.unit}"
  }

  tags = [ "${var.http_check_tags}" ]
}
`
