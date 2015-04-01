package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/monitors"
)

func TestAccLBV1Monitor_basic(t *testing.T) {
	var monitor monitors.Monitor

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1MonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Monitor_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV1MonitorExists(t, "openstack_lb_monitor_v1.monitor_1", &monitor),
				),
			},
			resource.TestStep{
				Config: testAccLBV1Monitor_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_monitor_v1.monitor_1", "delay", "20"),
				),
			},
		},
	})
}

func testAccCheckLBV1MonitorDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckLBV1MonitorDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_monitor_v1" {
			continue
		}

		_, err := monitors.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("LB monitor still exists")
		}
	}

	return nil
}

func testAccCheckLBV1MonitorExists(t *testing.T, n string, monitor *monitors.Monitor) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckLBV1MonitorExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := monitors.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Monitor not found")
		}

		*monitor = *found

		return nil
	}
}

var testAccLBV1Monitor_basic = fmt.Sprintf(`
  resource "openstack_lb_monitor_v1" "monitor_1" {
    region = "%s"
    type = "PING"
    delay = 30
    timeout = 5
    max_retries = 3
    admin_state_up = "true"
  }`,
	OS_REGION_NAME)

var testAccLBV1Monitor_update = fmt.Sprintf(`
  resource "openstack_lb_monitor_v1" "monitor_1" {
    region = "%s"
    type = "PING"
    delay = 20
    timeout = 5
    max_retries = 3
    admin_state_up = "true"
  }`,
	OS_REGION_NAME)
