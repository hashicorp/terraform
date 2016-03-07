package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
)

func TestAccNetworkingV2Router_basic(t *testing.T) {
	var router routers.Router

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2RouterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Router_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2RouterExists(t, "openstack_networking_router_v2.foo", &router),
				),
			},
			resource.TestStep{
				Config: testAccNetworkingV2Router_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_networking_router_v2.foo", "name", "router_2"),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2RouterDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckNetworkingV2RouterDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_router_v2" {
			continue
		}

		_, err := routers.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Router still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2RouterExists(t *testing.T, n string, router *routers.Router) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2RouterExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := routers.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Router not found")
		}

		*router = *found

		return nil
	}
}

var testAccNetworkingV2Router_basic = fmt.Sprintf(`
  resource "openstack_networking_router_v2" "foo" {
    name = "router"
    admin_state_up = "true"
    distributed = "false"
  }`)

var testAccNetworkingV2Router_update = fmt.Sprintf(`
  resource "openstack_networking_router_v2" "foo" {
    name = "router_2"
    admin_state_up = "true"
    distributed = "false"
  }`)
