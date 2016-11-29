package openstack

import (
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/pools"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccLBV2Member_basic(t *testing.T) {
	var member pools.Member

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV2MemberDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: TestAccLBV2MemberConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV2MemberExists(t, "openstack_lb_member_v2.member_1", &member),
				),
			},
			resource.TestStep{
				Config: TestAccLBV2MemberConfig_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_member_v2.member_1", "weight", "10"),
				),
			},
		},
	})
}

func testAccCheckLBV2MemberDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckLBV2MemberDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_member_v2" {
			continue
		}

		_, err := pools.GetMember(networkingClient, rs.Primary.Attributes["pool_id"], rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Member still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckLBV2MemberExists(t *testing.T, n string, member *pools.Member) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckLBV2MemberExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := pools.GetMember(networkingClient, rs.Primary.Attributes["pool_id"], rs.Primary.ID).Extract()
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

var TestAccLBV2MemberConfig_basic = fmt.Sprintf(`
	resource "openstack_networking_network_v2" "network_1" {
		name = "tf_test_network"
		admin_state_up = "true"
	}

	resource "openstack_networking_subnet_v2" "subnet_1" {
		network_id = "${openstack_networking_network_v2.network_1.id}"
		cidr = "192.168.199.0/24"
		ip_version = 4
		name = "tf_test_subnet"
	}

	resource "openstack_lb_loadbalancer_v2" "loadbalancer_1" {
		vip_subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
		name = "tf_test_loadbalancer_v2"
	}

	resource "openstack_lb_listener_v2" "listener_1" {
		protocol = "HTTP"
		protocol_port = 8080
		loadbalancer_id = "${openstack_lb_loadbalancer_v2.loadbalancer_1.id}"
		name = "tf_test_listener"
	}

	resource "openstack_lb_pool_v2" "pool_1" {
		protocol = "HTTP"
		lb_method = "ROUND_ROBIN"
		listener_id = "${openstack_lb_listener_v2.listener_1.id}"
		name = "tf_test_pool"
	}

	resource "openstack_lb_member_v2" "member_1" {
		address = "192.168.199.10"
		pool_id = "${openstack_lb_pool_v2.pool_1.id}"
		protocol_port = 8080
		subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
	}`)

var TestAccLBV2MemberConfig_update = fmt.Sprintf(`
	resource "openstack_networking_network_v2" "network_1" {
		name = "tf_test_network"
		admin_state_up = "true"
	}

	resource "openstack_networking_subnet_v2" "subnet_1" {
		network_id = "${openstack_networking_network_v2.network_1.id}"
		cidr = "192.168.199.0/24"
		ip_version = 4
		name = "tf_test_subnet"
	}

	resource "openstack_lb_loadbalancer_v2" "loadbalancer_1" {
		vip_subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
		name = "tf_test_loadbalancer_v2"
	}

	resource "openstack_lb_listener_v2" "listener_1" {
		protocol = "HTTP"
		protocol_port = 8080
		loadbalancer_id = "${openstack_lb_loadbalancer_v2.loadbalancer_1.id}"
		name = "tf_test_listener"
	}

	resource "openstack_lb_pool_v2" "pool_1" {
		protocol = "HTTP"
		lb_method = "ROUND_ROBIN"
		listener_id = "${openstack_lb_listener_v2.listener_1.id}"
		name = "tf_test_pool"
	}

	resource "openstack_lb_member_v2" "member_1" {
		address = "192.168.199.10"
		pool_id = "${openstack_lb_pool_v2.pool_1.id}"
		protocol_port = 8080
		subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
		weight = 10
		admin_state_up = "true"
	}`)
