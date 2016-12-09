package openstack

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccLBV2LoadBalancer_basic(t *testing.T) {
	var lb loadbalancers.LoadBalancer

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLBV2LoadBalancerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: TestAccLBV2LoadBalancerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLBV2LoadBalancerExists(t, "openstack_lb_loadbalancer_v2.loadbalancer_1", &lb),
				),
			},
			resource.TestStep{
				Config: TestAccLBV2LoadBalancerConfig_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_lb_loadbalancer_v2.loadbalancer_1", "name", "tf_test_loadbalancer_v2_updated"),
					resource.TestMatchResourceAttr("openstack_lb_loadbalancer_v2.loadbalancer_1", "vip_port_id", regexp.MustCompile("^[a-f0-9-]+")),
				),
			},
		},
	})
}

func testAccCheckLBV2LoadBalancerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckLBV2LoadBalancerDestroy) Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_lb_loadbalancer_v2" {
			continue
		}

		_, err := loadbalancers.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("LoadBalancer still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckLBV2LoadBalancerExists(t *testing.T, n string, lb *loadbalancers.LoadBalancer) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckLBV2LoadBalancerExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := loadbalancers.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Member not found")
		}

		*lb = *found

		return nil
	}
}

var TestAccLBV2LoadBalancerConfig_basic = fmt.Sprintf(`
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
	}`)

var TestAccLBV2LoadBalancerConfig_update = fmt.Sprintf(`
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
		name = "tf_test_loadbalancer_v2_updated"
		admin_state_up = "true"
  }
`)
