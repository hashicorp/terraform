package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/vips"
)

func TestAccLBV1VIP_basic(t *testing.T) {
	var vip vips.VirtualIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1VIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1VIP_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV1VIPExists(t, "openstack_lb_vip_v1.vip_1", &vip),
				),
			},
			resource.TestStep{
				Config: testAccLBV1VIP_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_vip_v1.vip_1", "name", "tf_test_lb_vip_updated"),
				),
			},
		},
	})
}

func testAccCheckLBV1VIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckLBV1VIPDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_vip_v1" {
			continue
		}

		_, err := vips.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("LB VIP still exists")
		}
	}

	return nil
}

func testAccCheckLBV1VIPExists(t *testing.T, n string, vip *vips.VirtualIP) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckLBV1VIPExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := vips.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("VIP not found")
		}

		*vip = *found

		return nil
	}
}

var testAccLBV1VIP_basic = fmt.Sprintf(`
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
  }

  resource "openstack_lb_vip_v1" "vip_1" {
    region = "RegionOne"
    name = "tf_test_lb_vip"
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    protocol = "HTTP"
    port = 80
    pool_id = "${openstack_lb_pool_v1.pool_1.id}"
    admin_state_up = true
    persistence {
      type = "SOURCE_IP"
    }
  }`,
	OS_REGION_NAME, OS_REGION_NAME, OS_REGION_NAME)

var testAccLBV1VIP_update = fmt.Sprintf(`
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
  }

  resource "openstack_lb_vip_v1" "vip_1" {
    region = "RegionOne"
    name = "tf_test_lb_vip_updated"
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    protocol = "HTTP"
    port = 80
    pool_id = "${openstack_lb_pool_v1.pool_1.id}"
    persistence {
      type = "SOURCE_IP"
    }
    admin_state_up = true
  }`,
	OS_REGION_NAME, OS_REGION_NAME, OS_REGION_NAME)
