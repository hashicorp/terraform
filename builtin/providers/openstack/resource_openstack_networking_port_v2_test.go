package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func TestAccNetworkingV2Port_basic(t *testing.T) {
	var network networks.Network
	var port ports.Port
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2PortDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Port_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.foo", &subnet),
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.foo", &network),
					testAccCheckNetworkingV2PortExists(t, "openstack_networking_port_v2.foo", &port),
				),
			},
		},
	})
}

func TestAccNetworkingV2Port_noip(t *testing.T) {
	var network networks.Network
	var port ports.Port
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2PortDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Port_noip,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.foo", &subnet),
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.foo", &network),
					testAccCheckNetworkingV2PortExists(t, "openstack_networking_port_v2.foo", &port),
				),
			},
		},
	})
}

func TestAccNetworkingV2Port_allowedAddressPairs(t *testing.T) {
	var network networks.Network
	var subnet subnets.Subnet
	var vrrp_port, instance_port ports.Port

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2PortDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Port_allowedAddressPairs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.vrrp_subnet", &subnet),
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.vrrp_network", &network),
					testAccCheckNetworkingV2PortExists(t, "openstack_networking_port_v2.vrrp_port", &vrrp_port),
					testAccCheckNetworkingV2PortExists(t, "openstack_networking_port_v2.instance_port", &instance_port),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2PortDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckNetworkingV2PortDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_port_v2" {
			continue
		}

		_, err := ports.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Port still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2PortExists(t *testing.T, n string, port *ports.Port) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2PortExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := ports.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Port not found")
		}

		*port = *found

		return nil
	}
}

var testAccNetworkingV2Port_basic = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "foo" {
			name = "network_1"
			admin_state_up = "true"
		}

		resource "openstack_networking_subnet_v2" "foo" {
			name = "subnet_1"
			network_id = "${openstack_networking_network_v2.foo.id}"
			cidr = "192.168.199.0/24"
			ip_version = 4
		}

		resource "openstack_networking_port_v2" "foo" {
			name = "port_1"
			network_id = "${openstack_networking_network_v2.foo.id}"
			admin_state_up = "true"
			fixed_ip {
				subnet_id =  "${openstack_networking_subnet_v2.foo.id}"
				ip_address = "192.168.199.23"
			}
		}`)

var testAccNetworkingV2Port_noip = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "foo" {
			name = "network_1"
			admin_state_up = "true"
		}
		resource "openstack_networking_subnet_v2" "foo" {
			name = "subnet_1"
			network_id = "${openstack_networking_network_v2.foo.id}"
			cidr = "192.168.199.0/24"
			ip_version = 4
		}
		resource "openstack_networking_port_v2" "foo" {
			name = "port_1"
			network_id = "${openstack_networking_network_v2.foo.id}"
			admin_state_up = "true"
			fixed_ip {
				subnet_id =  "${openstack_networking_subnet_v2.foo.id}"
			}
		}`)

var testAccNetworkingV2Port_allowedAddressPairs = fmt.Sprintf(`
		resource "openstack_networking_network_v2" "vrrp_network" {
			name = "vrrp_network"
			admin_state_up = "true"
		}

		resource "openstack_networking_subnet_v2" "vrrp_subnet" {
			name = "vrrp_subnet"
			network_id = "${openstack_networking_network_v2.vrrp_network.id}"
			cidr = "10.0.0.0/24"
			ip_version = 4

			allocation_pools {
				start = "10.0.0.2"
				end = "10.0.0.200"
			}
		}

		resource "openstack_networking_router_v2" "vrrp_router" {
			name = "vrrp_router"
		}

		resource "openstack_networking_router_interface_v2" "vrrp_interface" {
			router_id = "${openstack_networking_router_v2.vrrp_router.id}"
			subnet_id = "${openstack_networking_subnet_v2.vrrp_subnet.id}"
		}

		resource "openstack_networking_port_v2" "vrrp_port" {
			name = "vrrp_port"
			network_id = "${openstack_networking_network_v2.vrrp_network.id}"
			admin_state_up = "true"
			fixed_ip {
				subnet_id =  "${openstack_networking_subnet_v2.vrrp_subnet.id}"
				ip_address = "10.0.0.201"
			}
		}

		resource "openstack_networking_port_v2" "instance_port" {
			name = "instance_port"
			network_id = "${openstack_networking_network_v2.vrrp_network.id}"
			admin_state_up = "true"

			allowed_address_pairs {
				ip_address = "${openstack_networking_port_v2.vrrp_port.fixed_ip.0.ip_address}"
				mac_address = "${openstack_networking_port_v2.vrrp_port.mac_address}"
			}
		}`)
