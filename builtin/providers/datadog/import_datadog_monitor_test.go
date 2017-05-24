package datadog

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestDatadogMonitor_import(t *testing.T) {
	resourceName := "datadog_monitor.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogMonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogMonitorConfigImported,
			},
			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccCheckDatadogMonitorConfigImported = `
resource "datadog_monitor" "foo" {
  name = "name for monitor foo"
  type = "metric alert"
  message = "some message Notify: @hipchat-channel"
  escalation_message = "the situation has escalated @pagerduty"

  query = "avg(last_1h):avg:aws.ec2.cpu{environment:foo,host:foo} by {host} > 2.5"

  thresholds {
    ok = 1.5
    warning = 2.3
    critical = 2.5
  }

  notify_no_data = false
  new_host_delay = 600
  renotify_interval = 60

  notify_audit = false
  timeout_h = 60
  include_tags = true
  require_full_window = true
  locked = false
  tags = ["foo:bar", "bar:baz"]
}
`
