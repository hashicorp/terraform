package datadog

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

func TestAccDatadogMetricAlert_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogMetricAlertDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogMetricAlertConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogMetricAlertExists("datadog_metric_alert.foo"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "name", "name for metric_alert foo"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "message", "description for metric_alert foo"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "metric", "aws.ec2.cpu"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "tags.0", "environment:foo"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "tags.1", "host:foo"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "keys.0", "host"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "keys.#", "1"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "time_aggr", "avg"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "time_window", "last_1h"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "space_aggr", "avg"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "operator", "<"),
					resource.TestCheckResourceAttr(
						"datadog_metric_alert.foo", "notify_no_data", "false"),
					// TODO: add warning and critical
				),
			},
		},
	})
}

func testAccCheckDatadogMetricAlertDestroy(s *terraform.State) error {

	client := testAccProvider.Meta().(*datadog.Client)
	for _, rs := range s.RootModule().Resources {
		for _, v := range strings.Split(rs.Primary.ID, "__") {
			if v == "" {
				fmt.Printf("Could not parse IDs. %s", v)
				return fmt.Errorf("Id not set.")
			}
			ID, iErr := strconv.Atoi(v)

			if iErr != nil {
				fmt.Printf("Received error converting string %s", iErr)
				return iErr
			}
			_, err := client.GetMonitor(ID)
			if err != nil {
				// 404 is what we want, anything else is an error. Sadly our API will return a string like so:
				// return errors.New("API error: " + resp.Status)
				// For now we'll use unfold :|
				if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
					continue
				} else {
					fmt.Errorf("Received an error retrieving monitor %s", err)
				}
			} else {
				fmt.Errorf("Monitor still exists. %s", err)
			}
		}
	}
	return nil
}

func testAccCheckDatadogMetricAlertExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*datadog.Client)
		for _, rs := range s.RootModule().Resources {
			for _, v := range strings.Split(rs.Primary.ID, "__") {
				if v == "" {
					fmt.Printf("Could not parse IDs. %s", v)
					return fmt.Errorf("Id not set.")
				}
				ID, iErr := strconv.Atoi(v)

				if iErr != nil {
					return fmt.Errorf("Received error converting string %s", iErr)
				}
				_, err := client.GetMonitor(ID)
				if err != nil {
					return fmt.Errorf("Received an error retrieving monitor %s", err)
				}
			}
		}
		return nil
	}
}

const testAccCheckDatadogMetricAlertConfigBasic = `
resource "datadog_metric_alert" "foo" {
  name = "name for metric_alert foo"
  message = "description for metric_alert foo"

  metric = "aws.ec2.cpu"
  tags = ["environment:foo", "host:foo"]
  keys = ["host"]

  time_aggr = "avg" // avg, sum, max, min, change, or pct_change
  time_window = "last_1h" // last_#m (5, 10, 15, 30), last_#h (1, 2, 4), or last_1d
  space_aggr = "avg" // avg, sum, min, or max
  operator = "<" // <, <=, >, >=, ==, or !=

  warning {
    threshold = 0
    notify = "@hipchat-<name>"
  }

  critical {
    threshold = 0
    notify = "@pagerduty"
  }

  notify_no_data = false

}
`
