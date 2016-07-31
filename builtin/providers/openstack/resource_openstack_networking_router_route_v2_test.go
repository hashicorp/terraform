package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
)

func TestAccNetworkingV2RouterRoute_basic(t *testing.T) {
	var router routers.Router
	var network [2]networks.Network
	var subnet [2]subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2RouterRoute_create,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2RouterExists(t, "openstack_networking_router_v2.router_1", &router),
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.network_1", &network[0]),
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.subnet_1", &subnet[0]),
					testAccCheckNetworkingV2NetworkExists(t, "openstack_networking_network_v2.network_1", &network[1]),
					testAccCheckNetworkingV2SubnetExists(t, "openstack_networking_subnet_v2.subnet_1", &subnet[1]),
					testAccCheckNetworkingV2RouterInterfaceExists(t, "openstack_networking_router_interface_v2.int_1"),
					testAccCheckNetworkingV2RouterInterfaceExists(t, "openstack_networking_router_interface_v2.int_2"),
					testAccCheckNetworkingV2RouterRouteExists(t, "openstack_networking_router_route_v2.router_route_1"),
				),
			},
			resource.TestStep{
				Config: testAccNetworkingV2RouterRoute_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2RouterRouteExists(t, "openstack_networking_router_route_v2.router_route_1"),
					testAccCheckNetworkingV2RouterRouteExists(t, "openstack_networking_router_route_v2.router_route_2"),
				),
			},
			resource.TestStep{
				Config: testAccNetworkingV2RouterRoute_destroy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2RouterRouteEmpty(t, "openstack_networking_router_v2.router_1"),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2RouterRouteEmpty(t *testing.T, n string) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2RouterRouteExists) Error creating OpenStack networking client: %s", err)
		}

		router, err := routers.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if router.ID != rs.Primary.ID {
			return fmt.Errorf("Router not found")
		}

		if len(router.Routes) != 0 {
			return fmt.Errorf("Invalid number of route entries: %d", len(router.Routes))
		}

		return nil
	}
}

func testAccCheckNetworkingV2RouterRouteExists(t *testing.T, n string) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2RouterRouteExists) Error creating OpenStack networking client: %s", err)
		}

		router, err := routers.Get(networkingClient, rs.Primary.Attributes["router_id"]).Extract()
		if err != nil {
			return err
		}

		if router.ID != rs.Primary.Attributes["router_id"] {
			return fmt.Errorf("Router for route not found")
		}

		var found bool = false
		for _, r := range router.Routes {
			if r.DestinationCIDR == rs.Primary.Attributes["destination_cidr"] && r.NextHop == rs.Primary.Attributes["next_hop"] {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Could not find route for destination CIDR: %s, next hop: %s", rs.Primary.Attributes["destination_cidr"], rs.Primary.Attributes["next_hop"])
		}

		return nil
	}
}

var testAccNetworkingV2RouterRoute_create = fmt.Sprintf(`
	resource "openstack_networking_router_v2" "router_1" {
		name = "router_1"
		admin_state_up = "true"
	}

	resource "openstack_networking_network_v2" "network_1" {
			name = "network_1"
			admin_state_up = "true"
	}

	resource "openstack_networking_subnet_v2" "subnet_1" {
			network_id = "${openstack_networking_network_v2.network_1.id}"
			cidr = "192.168.199.0/24"
			ip_version = 4
	}

        resource "openstack_networking_port_v2" "port_1" {
                name = "port_1"
                network_id = "${openstack_networking_network_v2.network_1.id}"
                admin_state_up = "true"
                fixed_ip {
                        subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
                        ip_address = "192.168.199.1"
                }
        }

        resource "openstack_networking_router_interface_v2" "int_1" {
                        router_id = "${openstack_networking_router_v2.router_1.id}"
                        port_id = "${openstack_networking_port_v2.port_1.id}"
        }

        resource "openstack_networking_network_v2" "network_2" {
                        name = "network_2"
                        admin_state_up = "true"
        }

        resource "openstack_networking_subnet_v2" "subnet_2" {
                        network_id = "${openstack_networking_network_v2.network_2.id}"
                        cidr = "192.168.200.0/24"
                        ip_version = 4
        }

        resource "openstack_networking_port_v2" "port_2" {
                name = "port_2"
                network_id = "${openstack_networking_network_v2.network_2.id}"
                admin_state_up = "true"
                fixed_ip {
                        subnet_id = "${openstack_networking_subnet_v2.subnet_2.id}"
                        ip_address = "192.168.200.1"
                }
        }

        resource "openstack_networking_router_interface_v2" "int_2" {
                        router_id = "${openstack_networking_router_v2.router_1.id}"
                        port_id = "${openstack_networking_port_v2.port_2.id}"
        }

	resource "openstack_networking_router_route_v2" "router_route_1" {
			depends_on = ["openstack_networking_router_interface_v2.int_1"]
			router_id = "${openstack_networking_router_v2.router_1.id}"

			destination_cidr = "10.0.1.0/24"
			next_hop = "192.168.199.254"
	}`)

var testAccNetworkingV2RouterRoute_update = fmt.Sprintf(`
	resource "openstack_networking_router_v2" "router_1" {
		name = "router_1"
		admin_state_up = "true"
	}

	resource "openstack_networking_network_v2" "network_1" {
			name = "network_1"
			admin_state_up = "true"
	}

	resource "openstack_networking_subnet_v2" "subnet_1" {
			network_id = "${openstack_networking_network_v2.network_1.id}"
			cidr = "192.168.199.0/24"
			ip_version = 4
	}

        resource "openstack_networking_port_v2" "port_1" {
                name = "port_1"
                network_id = "${openstack_networking_network_v2.network_1.id}"
                admin_state_up = "true"
                fixed_ip {
                        subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
                        ip_address = "192.168.199.1"
                }
        }

        resource "openstack_networking_router_interface_v2" "int_1" {
                        router_id = "${openstack_networking_router_v2.router_1.id}"
                        port_id = "${openstack_networking_port_v2.port_1.id}"
        }

        resource "openstack_networking_network_v2" "network_2" {
                        name = "network_2"
                        admin_state_up = "true"
        }

        resource "openstack_networking_subnet_v2" "subnet_2" {
                        network_id = "${openstack_networking_network_v2.network_2.id}"
                        cidr = "192.168.200.0/24"
                        ip_version = 4
        }

        resource "openstack_networking_port_v2" "port_2" {
                name = "port_2"
                network_id = "${openstack_networking_network_v2.network_2.id}"
                admin_state_up = "true"
                fixed_ip {
                        subnet_id = "${openstack_networking_subnet_v2.subnet_2.id}"
                        ip_address = "192.168.200.1"
                }
        }

        resource "openstack_networking_router_interface_v2" "int_2" {
                        router_id = "${openstack_networking_router_v2.router_1.id}"
                        port_id = "${openstack_networking_port_v2.port_2.id}"
        }

	resource "openstack_networking_router_route_v2" "router_route_1" {
			depends_on = ["openstack_networking_router_interface_v2.int_1"]
			router_id = "${openstack_networking_router_v2.router_1.id}"

			destination_cidr = "10.0.1.0/24"
			next_hop = "192.168.199.254"
	}

	resource "openstack_networking_router_route_v2" "router_route_2" {
			depends_on = ["openstack_networking_router_interface_v2.int_2"]
			router_id = "${openstack_networking_router_v2.router_1.id}"

			destination_cidr = "10.0.2.0/24"
			next_hop = "192.168.200.254"
	}`)

var testAccNetworkingV2RouterRoute_destroy = fmt.Sprintf(`
	resource "openstack_networking_router_v2" "router_1" {
		name = "router_1"
		admin_state_up = "true"
	}

	resource "openstack_networking_network_v2" "network_1" {
			name = "network_1"
			admin_state_up = "true"
	}

	resource "openstack_networking_subnet_v2" "subnet_1" {
			network_id = "${openstack_networking_network_v2.network_1.id}"
			cidr = "192.168.199.0/24"
			ip_version = 4
	}

        resource "openstack_networking_port_v2" "port_1" {
                name = "port_1"
                network_id = "${openstack_networking_network_v2.network_1.id}"
                admin_state_up = "true"
                fixed_ip {
                        subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
                        ip_address = "192.168.199.1"
                }
        }

        resource "openstack_networking_router_interface_v2" "int_1" {
                        router_id = "${openstack_networking_router_v2.router_1.id}"
                        port_id = "${openstack_networking_port_v2.port_1.id}"
        }

        resource "openstack_networking_network_v2" "network_2" {
                        name = "network_2"
                        admin_state_up = "true"
        }

        resource "openstack_networking_subnet_v2" "subnet_2" {
                        network_id = "${openstack_networking_network_v2.network_2.id}"
                        cidr = "192.168.200.0/24"
                        ip_version = 4
        }

        resource "openstack_networking_port_v2" "port_2" {
                name = "port_2"
                network_id = "${openstack_networking_network_v2.network_2.id}"
                admin_state_up = "true"
                fixed_ip {
                        subnet_id = "${openstack_networking_subnet_v2.subnet_2.id}"
                        ip_address = "192.168.200.1"
                }
        }

        resource "openstack_networking_router_interface_v2" "int_2" {
                        router_id = "${openstack_networking_router_v2.router_1.id}"
                        port_id = "${openstack_networking_port_v2.port_2.id}"
	}`)
