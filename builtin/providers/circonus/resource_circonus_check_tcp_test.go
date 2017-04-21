package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckTCP_basic(t *testing.T) {
	checkName := fmt.Sprintf("Terraform test: TCP+TLS check - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckTCPConfigFmt, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "collector.1893401625.id", "/broker/1286"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.453641246.banner_regexp", ""),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.453641246.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.453641246.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.453641246.ciphers", ""),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.453641246.host", "127.0.0.1"),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.453641246.key_file", ""),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.453641246.port", "443"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "notes", "Check to harvest cert expiration information"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.#", "9"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.name", "cert_end"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1245733907.unit", "epoch"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.name", "cert_end_in"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2000319022.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.name", "cert_error"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.280072942.unit", ""),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.name", "cert_issuer"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1101485564.unit", ""),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.name", "cert_start"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3987659273.unit", "epoch"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.name", "cert_subject"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3170432128.unit", ""),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.name", "duration"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3590989341.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.name", "tt_connect"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.208818063.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.name", "tt_firstbyte"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4054733260.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "target", "127.0.0.1"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "type", "tcp"),
				),
			},
		},
	})
}

const testAccCirconusCheckTCPConfigFmt = `
variable "tcp_check_tags" {
  type = "list"
  default = [ "app:circonus", "app:tls_cert", "lifecycle:unittest", "source:fastly" ]
}

resource "circonus_check" "tls_cert" {
  active = true
  name = "%s"
  notes = "Check to harvest cert expiration information"
  period = "60s"

  collector {
    id = "/broker/1286"
  }

  tcp {
    host = "127.0.0.1"
    port = 443
  }

  metric {
    name = "cert_end"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "epoch"
  }

  metric {
    name = "cert_end_in"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "cert_error"
    tags = [ "${var.tcp_check_tags}" ]
    type = "text"
  }

  metric {
    name = "cert_issuer"
    tags = [ "${var.tcp_check_tags}" ]
    type = "text"
  }

  metric {
    name = "cert_start"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "epoch"
  }

  metric {
    name = "cert_subject"
    tags = [ "${var.tcp_check_tags}" ]
    type = "text"
  }

  metric {
    name = "duration"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "milliseconds"
  }

  metric {
    name = "tt_connect"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "milliseconds"
  }

  metric {
    name = "tt_firstbyte"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "milliseconds"
  }

  tags = [ "${var.tcp_check_tags}" ]
}
`
