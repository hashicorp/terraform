package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/groups"
)

func TestAccNetworkingV2SecGroup_basic(t *testing.T) {
	var security_group groups.SecGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SecGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2SecGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SecGroupExists(t, "openstack_networking_secgroup_v2.foo", &security_group),
				),
			},
			resource.TestStep{
				Config: testAccNetworkingV2SecGroup_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_networking_secgroup_v2.foo", "name", "security_group_2"),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2SecGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckNetworkingV2SecGroupDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_secgroup_v2" {
			continue
		}

		_, err := groups.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Security group still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2SecGroupExists(t *testing.T, n string, security_group *groups.SecGroup) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckNetworkingV2SecGroupExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := groups.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Security group not found")
		}

		*security_group = *found

		return nil
	}
}

var testAccNetworkingV2SecGroup_basic = fmt.Sprintf(`
  resource "openstack_networking_secgroup_v2" "foo" {
    name = "security_group"
    description = "terraform security group acceptance test"
  }`)

var testAccNetworkingV2SecGroup_update = fmt.Sprintf(`
  resource "openstack_networking_secgroup_v2" "foo" {
    name = "security_group_2"
    description = "terraform security group acceptance test"
  }`)
