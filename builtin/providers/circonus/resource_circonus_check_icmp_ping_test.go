package circonus

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckICMPPing_basic(t *testing.T) {
	checkName := fmt.Sprintf("ICMP Ping check - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckICMPPingConfigFmt, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "active", "true"),
					resource.TestCheckNoResourceAttr("circonus_check.loopback_latency", "check_id"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "checks.#", "2"),
					resource.TestMatchResourceAttr("circonus_check.loopback_latency", "checks.0", regexp.MustCompile(config.CheckCIDRegex)),
					resource.TestMatchResourceAttr("circonus_check.loopback_latency", "checks.1", regexp.MustCompile(config.CheckCIDRegex)),
					resource.TestCheckNoResourceAttr("circonus_check.loopback_latency", "check_id"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "check_by_collector.%", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "collector.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "collector.2388330941.id", "/broker/1"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "icmp_ping.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "icmp_ping.979664239.availability", "100"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "icmp_ping.979664239.count", "5"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "icmp_ping.979664239.interval", "500ms"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "period", "300s"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.#", "5"),

					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.784357201.name", "available"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.784357201.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.784357201.tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.784357201.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.784357201.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.784357201.unit", "%"),

					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.3166992875.name", "average"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.3166992875.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.3166992875.tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.3166992875.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.3166992875.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.3166992875.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.809361245.name", "count"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.809361245.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.809361245.tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.809361245.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.809361245.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.809361245.unit", "packets"),

					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.839816201.name", "maximum"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.839816201.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.839816201.tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.839816201.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.839816201.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.839816201.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.1657693034.name", "minimum"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.1657693034.tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.1657693034.tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.1657693034.tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.1657693034.type", "numeric"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "metric.1657693034.unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "target", "api.circonus.com"),
					resource.TestCheckResourceAttr("circonus_check.loopback_latency", "type", "ping_icmp"),
				),
			},
		},
	})
}

const testAccCirconusCheckICMPPingConfigFmt = `
variable "test_tags" {
  type = "list"
  default = [ "author:terraform", "lifecycle:unittest" ]
}
resource "circonus_check" "loopback_latency" {
  active = true
  name = "%s"
  period = "300s"

  collector {
    id = "/broker/1"
  }

  collector {
    id = "/broker/275"
  }

  icmp_ping {
    availability = "100.0"
    count = 5
    interval = "500ms"
  }

  metric {
    name = "available"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "%%"
  }

  metric {
    name = "average"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "count"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "packets"
  }

  metric {
    name = "maximum"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  metric {
    name = "minimum"
    tags = [ "${var.test_tags}" ]
    type = "numeric"
    unit = "seconds"
  }

  tags = [ "${var.test_tags}" ]
  target = "api.circonus.com"
}
`
