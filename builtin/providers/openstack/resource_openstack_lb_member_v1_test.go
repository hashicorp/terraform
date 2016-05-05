package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/members"
)

func TestAccLBV1Member_basic(t *testing.T) {
	var member members.Member

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1MemberDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Member_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV1MemberExists(t, "openstack_lb_member_v1.member_1", &member),
				),
			},
			resource.TestStep{
				Config: testAccLBV1Member_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_member_v1.member_1", "admin_state_up", "false"),
				),
			},
		},
	})
}

func testAccCheckLBV1MemberDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckLBV1MemberDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_member_v1" {
			continue
		}

		_, err := members.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("LB Member still exists")
		}
	}

	return nil
}

func testAccCheckLBV1MemberExists(t *testing.T, n string, member *members.Member) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckLBV1MemberExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := members.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Member not found")
		}

		*member = *found

		return nil
	}
}

var testAccLBV1Member_basic = fmt.Sprintf(`
  resource "openstack_networking_network_v2" "network_1" {
    name = "network_1"
    admin_state_up = "true"
  }

  resource "openstack_networking_subnet_v2" "subnet_1" {
    network_id = "${openstack_networking_network_v2.network_1.id}"
    cidr = "192.168.199.0/24"
    ip_version = 4
  }

  resource "openstack_lb_pool_v1" "pool_1" {
    name = "tf_test_lb_pool"
    protocol = "HTTP"
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    lb_method = "ROUND_ROBIN"
  }

  resource "openstack_lb_member_v1" "member_1" {
    pool_id = "${openstack_lb_pool_v1.pool_1.id}"
    address = "192.168.199.10"
    port = 80
    admin_state_up = true
  }`)

var testAccLBV1Member_update = fmt.Sprintf(`
  resource "openstack_networking_network_v2" "network_1" {
    name = "network_1"
    admin_state_up = "true"
  }

  resource "openstack_networking_subnet_v2" "subnet_1" {
    network_id = "${openstack_networking_network_v2.network_1.id}"
    cidr = "192.168.199.0/24"
    ip_version = 4
  }

  resource "openstack_lb_pool_v1" "pool_1" {
    name = "tf_test_lb_pool"
    protocol = "HTTP"
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    lb_method = "ROUND_ROBIN"
  }

  resource "openstack_lb_member_v1" "member_1" {
    pool_id = "${openstack_lb_pool_v1.pool_1.id}"
    address = "192.168.199.10"
    port = 80
    admin_state_up = false
  }`)
