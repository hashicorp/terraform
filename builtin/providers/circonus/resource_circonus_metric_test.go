package circonus

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCirconusMetric_basic(t *testing.T) {
	metricAvgName := fmt.Sprintf("Average Ping Time - %s", acctest.RandString(5))
	metricMaxName := fmt.Sprintf("Maximum Ping Time - %s", acctest.RandString(5))
	metricMinName := fmt.Sprintf("Minimum Ping Time - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyMetric,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusMetricConfigFmt, metricAvgName, metricMaxName, metricMinName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "name", metricAvgName),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "active", "false"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.3241999189", "source:circonus"),

					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "name", metricMaxName),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "active", "true"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "tags.2087084518", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "tags.3241999189", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_minimum", "name", metricMinName),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_minimum", "active", "true"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_minimum", "tags.#", "0"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_minimum", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_minimum", "unit", ""),
				),
			},
		},
	})
}

func TestAccCirconusMetric_tagsets(t *testing.T) {
	metricName := fmt.Sprintf("foo - %s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyMetric,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCirconusMetricTagsFmt0, metricName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", metricName),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCirconusMetricTagsFmt1, metricName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", metricName),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCirconusMetricTagsFmt2, metricName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", metricName),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2693443894", "foo:baz"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCirconusMetricTagsFmt3, metricName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", metricName),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "3"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2693443894", "foo:baz"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1937518738", "foo:bur"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCirconusMetricTagsFmt4, metricName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", metricName),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2693443894", "foo:baz"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1937518738", "foo:bur"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2110890696", "foo:baz2"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCirconusMetricTagsFmt5, metricName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", metricName),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "3"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1937518738", "foo:bur"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2110890696", "foo:baz2"),
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

const testAccCirconusMetricConfigFmt = `
resource "circonus_metric" "icmp_ping_average" {
  name = "%s"
  active = false
  type = "numeric"
  unit = "seconds"

  tags = [
    "author:terraform",
    "source:circonus",
  ]
}

resource "circonus_metric" "icmp_ping_maximum" {
  name = "%s"
  active = true
  type = "numeric"
  unit = "seconds"

  tags = [
    "source:circonus",
    "author:terraform",
  ]
}

resource "circonus_metric" "icmp_ping_minimum" {
  name = "%s"
  type = "numeric"
}
`

const testAccCirconusMetricTagsFmt0 = `
resource "circonus_metric" "t" {
  name = "%s"
# tags = [
#    "foo:bar",
#    "foo:baz",
#    "foo:bur",
#    "foo:baz2"
# ]
  type = "numeric"
}
`

const testAccCirconusMetricTagsFmt1 = `
resource "circonus_metric" "t" {
  name = "%s"
  tags = [
    "foo:bar",
#    "foo:baz",
#    "foo:bur",
#    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTagsFmt2 = `
resource "circonus_metric" "t" {
  name = "%s"
  tags = [
    "foo:bar",
    "foo:baz",
#    "foo:bur",
#    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTagsFmt3 = `
resource "circonus_metric" "t" {
  name = "%s"
  tags = [
    "foo:bar",
    "foo:baz",
    "foo:bur",
#    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTagsFmt4 = `
resource "circonus_metric" "t" {
  name = "%s"
  tags = [
    "foo:bar",
    "foo:baz",
    "foo:bur",
    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTagsFmt5 = `
resource "circonus_metric" "t" {
  name = "%s"
  tags = [
    "foo:bar",
#    "foo:baz",
    "foo:bur",
    "foo:baz2"
  ]
  type = "numeric"
}
`
