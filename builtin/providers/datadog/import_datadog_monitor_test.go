package datadog

import (
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestDatadogMonitor_import(t *testing.T) {
	resourceName := "datadog_monitor.bar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogMonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogMonitorConfig,
			},
			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
