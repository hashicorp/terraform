package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
)

func TestAccNetworkingV2RouterInterface_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2RouterInterfaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2RouterInterface_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2RouterInterfaceExists(t, "openstack_networking_router_interface_v2.int_1"),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2RouterInterfaceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckNetworkingV2RouterInterfaceDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_router_interface_v2" {
			continue
		}

		_, err := ports.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Router interface still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2RouterInterfaceExists(t *testing.T, n string) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2RouterInterfaceExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := ports.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Router interface not found")
		}

		return nil
	}
}

var testAccNetworkingV2RouterInterface_basic = fmt.Sprintf(`
resource "openstack_networking_router_v2" "router_1" {
  name = "router_1"
  admin_state_up = "true"
}

resource "openstack_networking_router_interface_v2" "int_1" {
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    router_id = "${openstack_networking_router_v2.router_1.id}"
}

resource "openstack_networking_network_v2" "network_1" {
    name = "network_1"
    admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
    network_id = "${openstack_networking_network_v2.network_1.id}"
    cidr = "192.168.199.0/24"
    ip_version = 4
}`)
