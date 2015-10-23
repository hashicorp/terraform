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

func TestAccDatadogMonitor_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogMonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogMonitorConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogMonitorExists("datadog_monitor.bar"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "name", "name for monitor foo"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "message", "description for monitor foo"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "metric", "aws.ec2.cpu"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "metric_tags", "*"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "time_aggr", "avg"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "time_window", "last_1h"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "space_aggr", "avg"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "operator", "<"),
					resource.TestCheckResourceAttr(
						"datadog_monitor.foo", "notify_no_data", "false"),
					// TODO: add warning and critical
				),
			},
		},
	})
}

func testAccCheckDatadogMonitorDestroy(s *terraform.State) error {

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
					fmt.Errorf("Received an error retreieving monitor %s", err)
				}
			} else {
				fmt.Errorf("Monitor still exists. %s", err)
			}
		}
	}
	return nil
}

func testAccCheckDatadogMonitorExists(n string) resource.TestCheckFunc {
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

const testAccCheckDatadogMonitorConfigBasic = `
resource "datadog_monitor" "foo" {
  name = "name for monitor foo"
  message = "description for monitor foo"

  metric = "aws.ec2.cpu"
  metric_tags = "*" // one or more comma separated tags (defaults to *)

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
