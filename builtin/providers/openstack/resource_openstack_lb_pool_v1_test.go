package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/monitors"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/pools"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/vips"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
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
					testAccCheckLBV1PoolExists(t, "openstack_lb_pool_v1.pool_1", &pool),
				),
			},
			resource.TestStep{
				Config: testAccLBV1Pool_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_pool_v1.pool_1", "name", "tf_test_lb_pool_updated"),
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
				Config: testAccLBV1Pool_fullstack,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.network_1", &network),
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.subnet_1", &subnet),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.secgroup_1", &secgroup),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_2", &instance2),
					testAccCheckLBV1PoolExists(t, "openstack_lb_pool_v1.pool_1", &pool),
					testAccCheckLBV1MonitorExists(t, "openstack_lb_monitor_v1.monitor_1", &monitor),
					testAccCheckLBV1VIPExists(t, "openstack_lb_vip_v1.vip_1", &vip),
				),
			},
		},
	})
}

func testAccCheckLBV1PoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckLBV1PoolDestroy) Error creating OpenStack networking client: %s", err)
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

func testAccCheckLBV1PoolExists(t *testing.T, n string, pool *pools.Pool) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckLBV1PoolExists) Error creating OpenStack networking client: %s", err)
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

var testAccLBV1Pool_basic = fmt.Sprintf(`
  resource "openstack_networking_network_v2" "network_1" {
    region = "%s"
    name = "network_1"
    admin_state_up = "true"
  }

  resource "openstack_networking_subnet_v2" "subnet_1" {
    region = "%s"
    network_id = "${openstack_networking_network_v2.network_1.id}"
    cidr = "192.168.199.0/24"
    ip_version = 4
  }

  resource "openstack_lb_pool_v1" "pool_1" {
    region = "%s"
    name = "tf_test_lb_pool"
    protocol = "HTTP"
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    lb_method = "ROUND_ROBIN"
  }`,
	OS_REGION_NAME, OS_REGION_NAME, OS_REGION_NAME)

var testAccLBV1Pool_update = fmt.Sprintf(`
  resource "openstack_networking_network_v2" "network_1" {
    region = "%s"
    name = "network_1"
    admin_state_up = "true"
  }

  resource "openstack_networking_subnet_v2" "subnet_1" {
    region = "%s"
    network_id = "${openstack_networking_network_v2.network_1.id}"
    cidr = "192.168.199.0/24"
    ip_version = 4
  }

  resource "openstack_lb_pool_v1" "pool_1" {
    region = "%s"
    name = "tf_test_lb_pool_updated"
    protocol = "HTTP"
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    lb_method = "ROUND_ROBIN"
  }`,
	OS_REGION_NAME, OS_REGION_NAME, OS_REGION_NAME)

var testAccLBV1Pool_fullstack = fmt.Sprintf(`
	resource "openstack_networking_network_v2" "network_1" {
		name = "network_1"
		admin_state_up = "true"
	}

	resource "openstack_networking_subnet_v2" "subnet_1" {
		network_id = "${openstack_networking_network_v2.network_1.id}"
		cidr = "192.168.199.0/24"
		ip_version = 4
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
		subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
		lb_method = "ROUND_ROBIN"
		monitor_ids = ["${openstack_lb_monitor_v1.monitor_1.id}"]
	}

	resource "openstack_lb_member_v1" "member_1" {
		pool_id = "${openstack_lb_pool_v1.pool_1.id}"
		address = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
		port = 80
		admin_state_up = true
	}

	resource "openstack_lb_member_v1" "member_2" {
		pool_id = "${openstack_lb_pool_v1.pool_1.id}"
		address = "${openstack_compute_instance_v2.instance_2.access_ip_v4}"
		port = 80
		admin_state_up = true
	}

	resource "openstack_lb_vip_v1" "vip_1" {
		name = "vip_1"
		subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
		protocol = "TCP"
		port = 80
		pool_id = "${openstack_lb_pool_v1.pool_1.id}"
		admin_state_up = true
	}`)
