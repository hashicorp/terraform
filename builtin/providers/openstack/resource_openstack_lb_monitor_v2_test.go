package openstack

import (
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/monitors"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccLBV2Monitor_basic(t *testing.T) {
	var monitor monitors.Monitor

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV2MonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: TestAccLBV2MonitorConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV2MonitorExists(t, "openstack_lb_monitor_v2.monitor_1", &monitor),
				),
			},
			resource.TestStep{
				Config: TestAccLBV2MonitorConfig_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"openstack_lb_monitor_v2.monitor_1", "name", "monitor_1_updated"),
					resource.TestCheckResourceAttr("openstack_lb_monitor_v2.monitor_1", "delay", "30"),
					resource.TestCheckResourceAttr("openstack_lb_monitor_v2.monitor_1", "timeout", "15"),
				),
			},
		},
	})
}

func TestAccLBV2Monitor_timeout(t *testing.T) {
	var monitor monitors.Monitor

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV2MonitorDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: TestAccLBV2MonitorConfig_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV2MonitorExists(t, "openstack_lb_monitor_v2.monitor_1", &monitor),
				),
			},
		},
	})
}

func testAccCheckLBV2MonitorDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_monitor_v2" {
			continue
		}

		_, err := monitors.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Monitor still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckLBV2MonitorExists(t *testing.T, n string, monitor *monitors.Monitor) resource.TestCheckFunc {
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
			return fmt.Errorf("Error creating OpenStack networking client: %s", err)
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

const TestAccLBV2MonitorConfig_basic = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_loadbalancer_v2" "loadbalancer_1" {
  name = "loadbalancer_1"
  vip_subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}

resource "openstack_lb_listener_v2" "listener_1" {
  name = "listener_1"
  protocol = "HTTP"
  protocol_port = 8080
  loadbalancer_id = "${openstack_lb_loadbalancer_v2.loadbalancer_1.id}"
}

resource "openstack_lb_pool_v2" "pool_1" {
  name = "pool_1"
  protocol = "HTTP"
  lb_method = "ROUND_ROBIN"
  listener_id = "${openstack_lb_listener_v2.listener_1.id}"
}

resource "openstack_lb_monitor_v2" "monitor_1" {
  name = "monitor_1"
  type = "PING"
  delay = 20
  timeout = 10
  max_retries = 5
  pool_id = "${openstack_lb_pool_v2.pool_1.id}"
}
`

const TestAccLBV2MonitorConfig_update = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_loadbalancer_v2" "loadbalancer_1" {
  name = "loadbalancer_1"
  vip_subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}

resource "openstack_lb_listener_v2" "listener_1" {
  name = "listener_1"
  protocol = "HTTP"
  protocol_port = 8080
  loadbalancer_id = "${openstack_lb_loadbalancer_v2.loadbalancer_1.id}"
}

resource "openstack_lb_pool_v2" "pool_1" {
  name = "pool_1"
  protocol = "HTTP"
  lb_method = "ROUND_ROBIN"
  listener_id = "${openstack_lb_listener_v2.listener_1.id}"
}

resource "openstack_lb_monitor_v2" "monitor_1" {
  name = "monitor_1_updated"
  type = "PING"
  delay = 30
  timeout = 15
  max_retries = 10
  admin_state_up = "true"
  pool_id = "${openstack_lb_pool_v2.pool_1.id}"
}
`

const TestAccLBV2MonitorConfig_timeout = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_loadbalancer_v2" "loadbalancer_1" {
  name = "loadbalancer_1"
  vip_subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}

resource "openstack_lb_listener_v2" "listener_1" {
  name = "listener_1"
  protocol = "HTTP"
  protocol_port = 8080
  loadbalancer_id = "${openstack_lb_loadbalancer_v2.loadbalancer_1.id}"
}

resource "openstack_lb_pool_v2" "pool_1" {
  name = "pool_1"
  protocol = "HTTP"
  lb_method = "ROUND_ROBIN"
  listener_id = "${openstack_lb_listener_v2.listener_1.id}"
}

resource "openstack_lb_monitor_v2" "monitor_1" {
  name = "monitor_1"
  type = "PING"
  delay = 20
  timeout = 10
  max_retries = 5
  pool_id = "${openstack_lb_pool_v2.pool_1.id}"

  timeouts {
    create = "5m"
    delete = "5m"
  }
}
`
