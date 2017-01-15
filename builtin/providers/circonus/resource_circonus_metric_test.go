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
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "active", "false"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.3051626963", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "tags.1384943139", "source:circonus"),

					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_average", "unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "name", "Maximum Ping Time"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "active", "true"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "tags.3051626963", "author:terraform"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "tags.1384943139", "source:circonus"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_maximum", "unit", "seconds"),

					resource.TestCheckResourceAttr("circonus_metric.icmp_ping_minimum", "name", "Minimum Ping Time"),
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
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDestroyMetric,
		Steps: []resource.TestStep{
			{
				Config: testAccCirconusMetricTags0,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", "foo"),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "0"),
				),
			},
			{
				Config: testAccCirconusMetricTags1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", "foo"),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "1"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
				),
			},
			{
				Config: testAccCirconusMetricTags2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", "foo"),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "2"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2693443894", "foo:baz"),
				),
			},
			{
				Config: testAccCirconusMetricTags3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", "foo"),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "3"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2693443894", "foo:baz"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1937518738", "foo:bur"),
				),
			},
			{
				Config: testAccCirconusMetricTags4,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", "foo"),
					resource.TestCheckResourceAttr("circonus_metric.t", "type", "numeric"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.#", "4"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1750285118", "foo:bar"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2693443894", "foo:baz"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.1937518738", "foo:bur"),
					resource.TestCheckResourceAttr("circonus_metric.t", "tags.2110890696", "foo:baz2"),
				),
			},
			{
				Config: testAccCirconusMetricTags5,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("circonus_metric.t", "name", "foo"),
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

const testAccCirconusMetricConfig = `
resource "circonus_metric" "icmp_ping_average" {
  name = "Average Ping Time"
  active = false
  type = "numeric"
  unit = "seconds"

  tags = [
    "author:terraform",
    "source:circonus",
  ]
}

resource "circonus_metric" "icmp_ping_maximum" {
  name = "Maximum Ping Time"
  active = true
  type = "numeric"
  unit = "seconds"

  tags = [
    "source:circonus",
    "author:terraform",
  ]
}

resource "circonus_metric" "icmp_ping_minimum" {
  name = "Minimum Ping Time"
  type = "numeric"
}
`

const testAccCirconusMetricTags0 = `
resource "circonus_metric" "t" {
  name = "foo"
# tags = [
#    "foo:bar",
#    "foo:baz",
#    "foo:bur",
#    "foo:baz2"
# ]
  type = "numeric"
}
`

const testAccCirconusMetricTags1 = `
resource "circonus_metric" "t" {
  name = "foo"
  tags = [
    "foo:bar",
#    "foo:baz",
#    "foo:bur",
#    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTags2 = `
resource "circonus_metric" "t" {
  name = "foo"
  tags = [
    "foo:bar",
    "foo:baz",
#    "foo:bur",
#    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTags3 = `
resource "circonus_metric" "t" {
  name = "foo"
  tags = [
    "foo:bar",
    "foo:baz",
    "foo:bur",
#    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTags4 = `
resource "circonus_metric" "t" {
  name = "foo"
  tags = [
    "foo:bar",
    "foo:baz",
    "foo:bur",
    "foo:baz2"
  ]
  type = "numeric"
}
`

const testAccCirconusMetricTags5 = `
resource "circonus_metric" "t" {
  name = "foo"
  tags = [
    "foo:bar",
#    "foo:baz",
    "foo:bur",
    "foo:baz2"
  ]
  type = "numeric"
}
`
