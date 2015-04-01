package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/pools"
)

func TestAccLBV1Pool_basic(t *testing.T) {
	var pool pools.Pool

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV1PoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLBV1Pool_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV1PoolExists(t, "openstack_lb_pool_v1.pool_1", &pool),
				),
			},
			resource.TestStep{
				Config: testAccLBV1Pool_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_pool_v1.pool_1", "name", "tf_test_lb_pool_updated"),
				),
			},
		},
	})
}

func testAccCheckLBV1PoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckLBV1PoolDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_pool_v1" {
			continue
		}

		_, err := pools.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("LB Pool still exists")
		}
	}

	return nil
}

func testAccCheckLBV1PoolExists(t *testing.T, n string, pool *pools.Pool) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckLBV1PoolExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := pools.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Pool not found")
		}

		*pool = *found

		return nil
	}
}

var testAccLBV1Pool_basic = fmt.Sprintf(`
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
  }`,
	OS_REGION_NAME, OS_REGION_NAME, OS_REGION_NAME)

var testAccLBV1Pool_update = fmt.Sprintf(`
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
    name = "tf_test_lb_pool_updated"
    protocol = "HTTP"
    subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
    lb_method = "ROUND_ROBIN"
  }`,
	OS_REGION_NAME, OS_REGION_NAME, OS_REGION_NAME)
