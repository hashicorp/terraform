package datadog

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

func TestAccDatadogOutlierAlert_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogOutlierAlertDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogOutlierAlertConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogOutlierAlertExists("datadog_outlier_alert.foo"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "name", "name for outlier_alert foo"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "message", "description for outlier_alert foo @hipchat-name"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "metric", "system.load.5"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "tags.0", "environment:foo"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "tags.1", "host:foo"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "keys.0", "host"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "keys.#", "1"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "time_aggr", "avg"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "time_window", "last_1h"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "space_aggr", "avg"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "notify_no_data", "false"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "algorithm", "mad"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "renotify_interval", "60"),
					resource.TestCheckResourceAttr(
						"datadog_outlier_alert.foo", "threshold", "2"),
				),
			},
		},
	})
}

func testAccCheckDatadogOutlierAlertDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)

	if err := destroyHelper(s, client); err != nil {
		return err
	}
	return nil
}

func testAccCheckDatadogOutlierAlertExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*datadog.Client)
		if err := existsHelper(s, client); err != nil {
			return err
		}
		return nil
	}
}

const testAccCheckDatadogOutlierAlertConfigBasic = `
resource "datadog_outlier_alert" "foo" {
  name = "name for outlier_alert foo"
  message = "description for outlier_alert foo @hipchat-name"

  algorithm = "mad"

  metric = "system.load.5"
  tags = ["environment:foo", "host:foo"]
  keys = ["host"]

  time_aggr = "avg" // avg, sum, max, min, change, or pct_change
  time_window = "last_1h" // last_#m (5, 10, 15, 30), last_#h (1, 2, 4), or last_1d
  space_aggr = "avg" // avg, sum, min, or max

  threshold = 2.0

  notify_no_data = false
  renotify_interval = 60

}
`
