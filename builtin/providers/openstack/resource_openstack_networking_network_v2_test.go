package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
)

func TestAccNetworkingV2Network_basic(t *testing.T) {
	region := os.Getenv(OS_REGION_NAME)

	var network networks.Network

	var testAccNetworkingV2Network_basic = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "foo" {
			region = "%s"
			name = "network_1"
			admin_state_up = "true"
		}`, region)

	var testAccNetworkingV2Network_update = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "foo" {
			region = "%s"
			name = "network_2"
			admin_state_up = "true"
		}`, region)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2NetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Network_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.foo", &network),
				),
			},
			resource.TestStep{
				Config: testAccNetworkingV2Network_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_networking_network_v2.foo", "name", "network_2"),
				),
			},
		},
	})
}

func TestAccNetworkingV2Network_netstack(t *testing.T) {
	region := os.Getenv(OS_REGION_NAME)

	var network networks.Network
	var subnet subnets.Subnet
	var router routers.Router

	var testAccNetworkingV2Network_netstack = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "foo" {
			region = "%s"
			name = "network_1"
			admin_state_up = "true"
		}

		resource "openstack_networking_subnet_v2" "foo" {
			region = "%s"
			name = "subnet_1"
			network_id = "${openstack_networking_network_v2.foo.id}"
			cidr = "192.168.10.0/24"
			ip_version = 4
		}

		resource "openstack_networking_router_v2" "foo" {
			region = "%s"
			name = "router_1"
		}

		resource "openstack_networking_router_interface_v2" "foo" {
			region = "%s"
			router_id = "${openstack_networking_router_v2.foo.id}"
			subnet_id = "${openstack_networking_subnet_v2.foo.id}"
		}`, region, region, region, region)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2NetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Network_netstack,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.foo", &network),
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.foo", &subnet),
					testAccCheckNetworkingV2RouterExists(t, "openstack_networking_router_v2.foo", &router),
					testAccCheckNetworkingV2RouterInterfaceExists(t, "openstack_networking_router_interface_v2.foo"),
				),
			},
		},
	})
}

func TestAccNetworkingV2Network_fullstack(t *testing.T) {
	region := os.Getenv(OS_REGION_NAME)

	var instance servers.Server
	var network networks.Network
	var port ports.Port
	var secgroup secgroups.SecurityGroup
	var subnet subnets.Subnet

	var testAccNetworkingV2Network_fullstack = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "foo" {
			region = "%s"
			name = "network_1"
			admin_state_up = "true"
		}

		resource "openstack_networking_subnet_v2" "foo" {
			region = "%s"
			name = "subnet_1"
			network_id = "${openstack_networking_network_v2.foo.id}"
			cidr = "192.168.199.0/24"
			ip_version = 4
		}

		resource "openstack_compute_secgroup_v2" "foo" {
			region = "%s"
			name = "secgroup_1"
			description = "a security group"
			rule {
				from_port = 22
				to_port = 22
				ip_protocol = "tcp"
				cidr = "0.0.0.0/0"
			}
		}

		resource "openstack_networking_port_v2" "foo" {
			region = "%s"
			name = "port_1"
			network_id = "${openstack_networking_network_v2.foo.id}"
			admin_state_up = "true"
			security_group_ids = ["${openstack_compute_secgroup_v2.foo.id}"]
			fixed_ip {
				"subnet_id" =  "${openstack_networking_subnet_v2.foo.id}"
				"ip_address" =  "192.168.199.23"
			}
		}

		resource "openstack_compute_instance_v2" "foo" {
			region = "%s"
			name = "terraform-test"
			security_groups = ["${openstack_compute_secgroup_v2.foo.name}"]

			network {
				port = "${openstack_networking_port_v2.foo.id}"
			}
		}`, region, region, region, region, region)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2NetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Network_fullstack,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.foo", &network),
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.foo", &subnet),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.foo", &secgroup),
					testAccCheckNetworkingV2PortExists(t, "openstack_networking_port_v2.foo", &port),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2NetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckNetworkingV2NetworkDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_network_v2" {
			continue
		}

		_, err := networks.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Network still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2NetworkExists(t *testing.T, n string, network *networks.Network) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2NetworkExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := networks.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Network not found")
		}

		*network = *found

		return nil
	}
}

var testAccNetworkingV2Network_basic = fmt.Sprintf(`
	resource "openstack_networking_network_v2" "foo" {
		region = "%s"
		name = "network_1"
		admin_state_up = "true"
	}`,
	OS_REGION_NAME)

var testAccNetworkingV2Network_update = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "foo" {
			region = "%s"
			name = "network_2"
			admin_state_up = "true"
			}`,
	OS_REGION_NAME)
