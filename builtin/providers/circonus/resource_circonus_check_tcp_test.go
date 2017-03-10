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

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.name", "cert_end"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.2951598908.unit", "epoch"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.name", "cert_end_in"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.4072382121.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.name", "cert_error"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3384170740.unit", ""),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.name", "cert_issuer"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.979255163.unit", ""),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.name", "cert_start"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1378403576.unit", "epoch"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.name", "cert_subject"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.1662016973.unit", ""),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.name", "duration"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.872453198.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.name", "tt_connect"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.719003215.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.name", "tt_firstbyte"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.tags.862116066", "source:fastly"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "metric.3321090683.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.213659730", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.1543130091", "lifecycle:unittests"),
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
  default = [ "app:circonus", "app:tls_cert", "lifecycle:unittests", "source:fastly" ]
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
