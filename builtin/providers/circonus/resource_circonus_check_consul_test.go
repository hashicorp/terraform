package circonus

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckConsul_node(t *testing.T) {
	checkName := fmt.Sprintf("Terraform test: consul.service.consul mode=state check - %s", acctest.RandString(5))

	checkNode := fmt.Sprintf("my-node-name-or-node-id-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckConsulConfigV1HealthNodeFmt, checkName, checkNode),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.consul_server", "active", "true"),
					resource.TestMatchResourceAttr("circonus_check.consul_server", "check_id", regexp.MustCompile(config.CheckCIDRegex)),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "collector.2084916526.id", "/broker/2110"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.ciphers", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.key_file", ""),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.dc", "dc2"),
					resource.TestCheckNoResourceAttr("circonus_check.consul_server", "consul.0.headers"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.http_addr", "http://consul.service.consul:8501"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.node", checkNode),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.node_blacklist.#", "3"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.node_blacklist.0", "a"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.node_blacklist.1", "bad"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.node_blacklist.2", "node"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "notes", ""),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.name", "KnownLeader"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.name", "LastContact"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "target", "consul.service.consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "type", "consul"),
				),
			},
		},
	})
}

func TestAccCirconusCheckConsul_service(t *testing.T) {
	checkName := fmt.Sprintf("Terraform test: consul.service.consul mode=service check - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckConsulConfigV1HealthServiceFmt, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.consul_server", "active", "true"),
					resource.TestMatchResourceAttr("circonus_check.consul_server", "check_id", regexp.MustCompile(config.CheckCIDRegex)),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "collector.2084916526.id", "/broker/2110"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.ciphers", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.key_file", ""),
					resource.TestCheckNoResourceAttr("circonus_check.consul_server", "consul.0.headers"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.http_addr", "http://consul.service.consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.service", "consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.service_blacklist.#", "3"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.service_blacklist.0", "bad"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.service_blacklist.1", "hombre"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.service_blacklist.2", "service"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "notes", ""),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.name", "KnownLeader"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.name", "LastContact"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "target", "consul.service.consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "type", "consul"),
				),
			},
		},
	})
}

func TestAccCirconusCheckConsul_state(t *testing.T) {
	checkName := fmt.Sprintf("Terraform test: consul.service.consul mode=state check - %s", acctest.RandString(5))

	checkState := "critical"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckConsulConfigV1HealthStateFmt, checkName, checkState),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.consul_server", "active", "true"),
					resource.TestMatchResourceAttr("circonus_check.consul_server", "check_id", regexp.MustCompile(config.CheckCIDRegex)),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "collector.2084916526.id", "/broker/2110"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.#", "1"),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.ca_chain", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.certificate_file", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.ciphers", ""),
					// resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.key_file", ""),
					resource.TestCheckNoResourceAttr("circonus_check.consul_server", "consul.0.headers"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.http_addr", "http://consul.service.consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.state", checkState),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.check_blacklist.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.check_blacklist.0", "worthless"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "consul.0.check_blacklist.1", "check"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "notes", ""),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.name", "KnownLeader"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3333874791.type", "text"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.active", "true"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.name", "LastContact"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "metric.3148913305.unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "tags.2058715988", "source:consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "target", "consul.service.consul"),
					resource.TestCheckResourceAttr("circonus_check.consul_server", "type", "consul"),
				),
			},
		},
	})
}

const testAccCirconusCheckConsulConfigV1HealthNodeFmt = `
resource "circonus_check" "consul_server" {
  active = true
  name = "%s"
  period = "60s"

  collector {
    id = "/broker/2110"
  }

  consul {
    dc = "dc2"
    http_addr = "http://consul.service.consul:8501"
    node = "%s"
    node_blacklist = ["a","bad","node"]
  }

  metric {
    name = "LastContact"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "KnownLeader"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "text"
  }

  tags = [ "source:consul", "lifecycle:unittest" ]

  target = "consul.service.consul"
}
`

const testAccCirconusCheckConsulConfigV1HealthServiceFmt = `
resource "circonus_check" "consul_server" {
  active = true
  name = "%s"
  period = "60s"

  collector {
    id = "/broker/2110"
  }

  consul {
    service = "consul"
    service_blacklist = ["bad","hombre","service"]
  }

  metric {
    name = "LastContact"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "KnownLeader"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "text"
  }

  tags = [ "source:consul", "lifecycle:unittest" ]

  target = "consul.service.consul"
}
`

const testAccCirconusCheckConsulConfigV1HealthStateFmt = `
resource "circonus_check" "consul_server" {
  active = true
  name = "%s"
  period = "60s"

  collector {
    id = "/broker/2110"
  }

  consul {
    state = "%s"
    check_blacklist = ["worthless","check"]
  }

  metric {
    name = "LastContact"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "KnownLeader"
    tags = [ "source:consul", "lifecycle:unittest" ]
    type = "text"
  }

  tags = [ "source:consul", "lifecycle:unittest" ]

  target = "consul.service.consul"
}
`
