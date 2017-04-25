package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas/monitors"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas/pools"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas/vips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func TestAccLBV1Pool_basic(t *testing.T) {
	var pool pools.Pool

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1PoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Pool_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV1PoolExists("openstack_lb_pool_v1.pool_1", &pool),
					resource.TestCheckResourceAttr("openstack_lb_pool_v1.pool_1", "lb_provider", "haproxy"),
				),
			},
			resource.TestStep{
				Config: testAccLBV1Pool_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_pool_v1.pool_1", "name", "pool_1"),
				),
			},
		},
	})
}

func TestAccLBV1Pool_fullstack(t *testing.T) {
	var instance1, instance2 servers.Server
	var monitor monitors.Monitor
	var network networks.Network
	var pool pools.Pool
	var secgroup secgroups.SecurityGroup
	var subnet subnets.Subnet
	var vip vips.VirtualIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1PoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Pool_fullstack_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists("openstack_networking_network_v2.network_1", &network),
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					testAccCheckComputeV2SecGroupExists("openstack_compute_secgroup_v2.secgroup_1", &secgroup),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance1),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_2", &instance2),
					testAccCheckLBV1PoolExists("openstack_lb_pool_v1.pool_1", &pool),
					testAccCheckLBV1MonitorExists("openstack_lb_monitor_v1.monitor_1", &monitor),
					testAccCheckLBV1VIPExists("openstack_lb_vip_v1.vip_1", &vip),
				),
			},
			resource.TestStep{
				Config: testAccLBV1Pool_fullstack_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists("openstack_networking_network_v2.network_1", &network),
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					testAccCheckComputeV2SecGroupExists("openstack_compute_secgroup_v2.secgroup_1", &secgroup),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance1),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_2", &instance2),
					testAccCheckLBV1PoolExists("openstack_lb_pool_v1.pool_1", &pool),
					testAccCheckLBV1MonitorExists("openstack_lb_monitor_v1.monitor_1", &monitor),
					testAccCheckLBV1VIPExists("openstack_lb_vip_v1.vip_1", &vip),
				),
			},
		},
	})
}

func TestAccLBV1Pool_timeout(t *testing.T) {
	var pool pools.Pool

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1PoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Pool_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV1PoolExists("openstack_lb_pool_v1.pool_1", &pool),
					resource.TestCheckResourceAttr("openstack_lb_pool_v1.pool_1", "lb_provider", "haproxy"),
				),
			},
		},
	})
}

func TestAccLBV1Pool_updateMonitor(t *testing.T) {
	var monitor_1 monitors.Monitor
	var monitor_2 monitors.Monitor
	var network networks.Network
	var pool pools.Pool
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1PoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Pool_updateMonitor_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists("openstack_networking_network_v2.network_1", &network),
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					testAccCheckLBV1PoolExists("openstack_lb_pool_v1.pool_1", &pool),
					testAccCheckLBV1MonitorExists("openstack_lb_monitor_v1.monitor_1", &monitor_1),
					testAccCheckLBV1MonitorExists("openstack_lb_monitor_v1.monitor_2", &monitor_2),
				),
			},
			resource.TestStep{
				Config: testAccLBV1Pool_updateMonitor_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists("openstack_networking_network_v2.network_1", &network),
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					testAccCheckLBV1PoolExists("openstack_lb_pool_v1.pool_1", &pool),
					testAccCheckLBV1MonitorExists("openstack_lb_monitor_v1.monitor_1", &monitor_1),
					testAccCheckLBV1MonitorExists("openstack_lb_monitor_v1.monitor_2", &monitor_2),
				),
			},
		},
	})
}

func testAccCheckLBV1PoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_pool_v1" {
			continue
		}

		_, err := pools.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("LB Pool still exists")
		}
	}

	return nil
}

func testAccCheckLBV1PoolExists(n string, pool *pools.Pool) resource.TestCheckFunc {
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

		found, err := pools.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Pool not found")
		}

		*pool = *found

		return nil
	}
}

const testAccLBV1Pool_basic = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name = "pool_1"
  protocol = "HTTP"
  lb_method = "ROUND_ROBIN"
  lb_provider = "haproxy"
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}
`

const testAccLBV1Pool_update = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name = "pool_1"
  protocol = "HTTP"
  lb_method = "ROUND_ROBIN"
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}
`

const testAccLBV1Pool_fullstack_1 = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "Rules for secgroup_1"

  rule {
    from_port = -1
    to_port = -1
    ip_protocol = "icmp"
    cidr = "0.0.0.0/0"
  }

  rule {
    from_port = 80
    to_port = 80
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}"]

  network {
    uuid = "${openstack_networking_network_v2.network_1.id}"
  }
}

resource "openstack_compute_instance_v2" "instance_2" {
  name = "instance_2"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}"]

  network {
    uuid = "${openstack_networking_network_v2.network_1.id}"
  }
}

resource "openstack_lb_monitor_v1" "monitor_1" {
  type = "TCP"
  delay = 30
  timeout = 5
  max_retries = 3
  admin_state_up = "true"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name = "pool_1"
  protocol = "TCP"
  lb_method = "ROUND_ROBIN"
  monitor_ids = ["${openstack_lb_monitor_v1.monitor_1.id}"]
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}

resource "openstack_lb_member_v1" "member_1" {
  port = 80
  admin_state_up = true
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
  address = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
}

resource "openstack_lb_member_v1" "member_2" {
  port = 80
  admin_state_up = true
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
  address = "${openstack_compute_instance_v2.instance_2.access_ip_v4}"
}

resource "openstack_lb_vip_v1" "vip_1" {
  name = "vip_1"
  protocol = "TCP"
  port = 80
  admin_state_up = true
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
}
`

const testAccLBV1Pool_fullstack_2 = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "Rules for secgroup_1"

  rule {
    from_port = -1
    to_port = -1
    ip_protocol = "icmp"
    cidr = "0.0.0.0/0"
  }

  rule {
    from_port = 80
    to_port = 80
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}"]

  network {
    uuid = "${openstack_networking_network_v2.network_1.id}"
  }
}

resource "openstack_compute_instance_v2" "instance_2" {
  name = "instance_2"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}"]
	user_data = "#cloud-config\ndisable_root: false"

  network {
    uuid = "${openstack_networking_network_v2.network_1.id}"
  }
}

resource "openstack_lb_monitor_v1" "monitor_1" {
  type = "TCP"
  delay = 30
  timeout = 5
  max_retries = 3
  admin_state_up = "true"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name = "pool_1"
  protocol = "TCP"
  lb_method = "ROUND_ROBIN"
  monitor_ids = ["${openstack_lb_monitor_v1.monitor_1.id}"]
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}

resource "openstack_lb_member_v1" "member_1" {
  port = 80
  admin_state_up = true
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
  address = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
}

resource "openstack_lb_member_v1" "member_2" {
  port = 80
  admin_state_up = true
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
  address = "${openstack_compute_instance_v2.instance_2.access_ip_v4}"
}

resource "openstack_lb_vip_v1" "vip_1" {
  name = "vip_1"
  protocol = "TCP"
  port = 80
  admin_state_up = true
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
}
`

const testAccLBV1Pool_timeout = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name = "pool_1"
  protocol = "HTTP"
  lb_method = "ROUND_ROBIN"
  lb_provider = "haproxy"
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"

  timeouts {
    create = "5m"
    delete = "5m"
  }
}
`

const testAccLBV1Pool_updateMonitor_1 = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_monitor_v1" "monitor_1" {
  type = "TCP"
  delay = 30
  timeout = 5
  max_retries = 3
  admin_state_up = "true"
}

resource "openstack_lb_monitor_v1" "monitor_2" {
  type = "TCP"
  delay = 30
  timeout = 5
  max_retries = 3
  admin_state_up = "true"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name = "pool_1"
  protocol = "TCP"
  lb_method = "ROUND_ROBIN"
  monitor_ids = ["${openstack_lb_monitor_v1.monitor_1.id}"]
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}
`

const testAccLBV1Pool_updateMonitor_2 = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  ip_version = 4
  network_id = "${openstack_networking_network_v2.network_1.id}"
}

resource "openstack_lb_monitor_v1" "monitor_1" {
  type = "TCP"
  delay = 30
  timeout = 5
  max_retries = 3
  admin_state_up = "true"
}

resource "openstack_lb_monitor_v1" "monitor_2" {
  type = "TCP"
  delay = 30
  timeout = 5
  max_retries = 3
  admin_state_up = "true"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name = "pool_1"
  protocol = "TCP"
  lb_method = "ROUND_ROBIN"
  monitor_ids = ["${openstack_lb_monitor_v1.monitor_2.id}"]
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}
`
