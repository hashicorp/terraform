package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCirconusCheckStatsd_basic(t *testing.T) {
	checkName := fmt.Sprintf("statsd test check - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyCirconusCheckBundle,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusCheckStatsdConfigFmt, checkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "active", "true"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "collector.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "collector.2084916526.id", "/broker/2110"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "statsd.#", "1"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "statsd.3733287963.source_ip", `127.0.0.2`),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "name", checkName),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "period", "60s"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "metric.#", "1"),

					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "tags.3728194417", "app:consul"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "tags.1401442048", "lifecycle:unittest"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "tags.2812916752", "source:statsd"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "target", "127.0.0.2"),
					resource.TestCheckResourceAttr("circonus_check.statsd_dump", "type", "statsd"),
				),
			},
		},
	})
}

const testAccCirconusCheckStatsdConfigFmt = `
variable "test_tags" {
  type = "list"
  default = [ "app:consul", "author:terraform", "lifecycle:unittest", "source:statsd" ]
}

resource "circonus_check" "statsd_dump" {
  active = true
  name = "%s"
  period = "60s"

  collector {
    id = "/broker/2110"
  }

  statsd {
    source_ip = "127.0.0.2"
  }

  metric {
    name = "rando_metric"
    tags = [ "${var.test_tags}" ]
    type = "histogram"
  }

  tags = [ "${var.test_tags}" ]
}
`
