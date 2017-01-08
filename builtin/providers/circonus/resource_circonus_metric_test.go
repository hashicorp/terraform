package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusMetric_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyMetric,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusMetricConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "name", "Average Ping Time"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "unit", "seconds"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.%", "2"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.author", "terraform"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.source", "circonus"),
				),
			},
		},
	})
}

func testAccCheckDestroyMetric(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "circonus_metric" {
			continue
		}

		id := rs.Primary.ID
		exists := id == ""
		switch {
		case !exists:
			// noop
		case exists:
			return fmt.Errorf("metric still exists after destroy")
		}
	}

	return nil
}

const testAccCirconusMetricConfig = `
resource "circonus_metric" "icmp_ping_average" {
  name = "Average Ping Time"
  type = "numeric"
  unit = "seconds"

  tags = {
    "author"= "terraform",
    "source"= "circonus",
  }
}
`
