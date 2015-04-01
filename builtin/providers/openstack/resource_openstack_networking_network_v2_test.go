package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
)

func TestAccNetworkingV2Network_basic(t *testing.T) {
	var network networks.Network

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
