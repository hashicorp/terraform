package circonus

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckTCP_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusCheckTCPConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "collector.1893401625.id", "/broker/1286"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.2201265081.banner_regexp", ""),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.2201265081.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.2201265081.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.2201265081.ciphers", ""),
					// resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.2201265081.key_file", ""),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tcp.2201265081.port", "443"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "name", "Terraform test: TLS check"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "notes", "Check to harvest cert expiration information"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.#", "9"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.name", "code"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.tags.3219687752", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.42262635.type", "text"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.name", "duration"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.tags.3219687752", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.1136493216.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.name", "tt_connect"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.tags.3219687752", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.4246441943.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.name", "tt_firstbyte"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.tags.3219687752", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "stream.3695203246.unit", "milliseconds"),

					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.30226350", "app:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.3219687752", "app:tls_cert"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.1543130091", "lifecycle:unittests"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "target", "127.0.0.1"),
					resource.TestCheckResourceAttr("circonus_check.tls_cert", "type", "tcp"),
				),
			},
		},
	})
}

const testAccCirconusCheckTCPConfig = `
variable "tcp_check_tags" {
  type = "list"
  default = [ "app:circonus", "app:tls_cert", "lifecycle:unittests", "source:fastly" ]
}

resource "circonus_check" "tls_cert" {
  active = true
  name = "Terraform test: TLS check"
  notes = "Check to harvest cert expiration information"
  period = "60s"

  collector {
    id = "/broker/1286"
  }

  tcp {
    port = 443
  }

  stream {
    name = "cert_end"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "epoch"
  }

  stream {
    name = "cert_end_in"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  stream {
    name = "cert_error"
    tags = [ "${var.tcp_check_tags}" ]
    type = "text"
  }

  stream {
    name = "cert_issuer"
    tags = [ "${var.tcp_check_tags}" ]
    type = "text"
  }

  stream {
    name = "cert_start"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "epoch"
  }

  stream {
    name = "cert_subject"
    tags = [ "${var.tcp_check_tags}" ]
    type = "text"
  }

  stream {
    name = "duration"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "milliseconds"
  }

  stream {
    name = "tt_connect"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "milliseconds"
  }

  stream {
    name = "tt_firstbyte"
    tags = [ "${var.tcp_check_tags}" ]
    type = "numeric"
    unit = "milliseconds"
  }

  tags = [ "${var.tcp_check_tags}" ]
}
`
