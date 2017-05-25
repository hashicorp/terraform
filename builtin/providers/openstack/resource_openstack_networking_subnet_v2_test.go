package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func TestAccNetworkingV2Subnet_basic(t *testing.T) {
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Subnet_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "allocation_pools.0.start", "192.168.199.100"),
				),
			},
			resource.TestStep{
				Config: testAccNetworkingV2Subnet_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "name", "subnet_1"),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "gateway_ip", "192.168.199.1"),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "enable_dhcp", "true"),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "allocation_pools.0.start", "192.168.199.150"),
				),
			},
		},
	})
}

func TestAccNetworkingV2Subnet_enableDHCP(t *testing.T) {
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Subnet_enableDHCP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "enable_dhcp", "true"),
				),
			},
		},
	})
}

func TestAccNetworkingV2Subnet_disableDHCP(t *testing.T) {
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Subnet_disableDHCP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "enable_dhcp", "false"),
				),
			},
		},
	})
}

func TestAccNetworkingV2Subnet_noGateway(t *testing.T) {
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Subnet_noGateway,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "gateway_ip", ""),
				),
			},
		},
	})
}

func TestAccNetworkingV2Subnet_impliedGateway(t *testing.T) {
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Subnet_impliedGateway,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
					resource.TestCheckResourceAttr(
						"openstack_networking_subnet_v2.subnet_1", "gateway_ip", "192.168.199.1"),
				),
			},
		},
	})
}

func TestAccNetworkingV2Subnet_timeout(t *testing.T) {
	var subnet subnets.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2SubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2Subnet_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2SubnetExists("openstack_networking_subnet_v2.subnet_1", &subnet),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2SubnetDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkingClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_subnet_v2" {
			continue
		}

		_, err := subnets.Get(networkingClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Subnet still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2SubnetExists(n string, subnet *subnets.Subnet) resource.TestCheckFunc {
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
			return fmt.Errorf("Error creating OpenStack networking client: %s", err)
		}

		found, err := subnets.Get(networkingClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Subnet not found")
		}

		*subnet = *found

		return nil
	}
}

const testAccNetworkingV2Subnet_basic = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  network_id = "${openstack_networking_network_v2.network_1.id}"

  allocation_pools {
    start = "192.168.199.100"
    end = "192.168.199.200"
  }
}
`

const testAccNetworkingV2Subnet_update = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  gateway_ip = "192.168.199.1"
  network_id = "${openstack_networking_network_v2.network_1.id}"

  allocation_pools {
    start = "192.168.199.150"
    end = "192.168.199.200"
  }
}
`

const testAccNetworkingV2Subnet_enableDHCP = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  gateway_ip = "192.168.199.1"
  enable_dhcp = true
  network_id = "${openstack_networking_network_v2.network_1.id}"
}
`

const testAccNetworkingV2Subnet_disableDHCP = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  enable_dhcp = false
  network_id = "${openstack_networking_network_v2.network_1.id}"
}
`

const testAccNetworkingV2Subnet_noGateway = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  no_gateway = true
  network_id = "${openstack_networking_network_v2.network_1.id}"
}
`

const testAccNetworkingV2Subnet_impliedGateway = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}
resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  cidr = "192.168.199.0/24"
  network_id = "${openstack_networking_network_v2.network_1.id}"
}
`

const testAccNetworkingV2Subnet_timeout = `
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  cidr = "192.168.199.0/24"
  network_id = "${openstack_networking_network_v2.network_1.id}"

  allocation_pools {
    start = "192.168.199.100"
    end = "192.168.199.200"
  }

  timeouts {
    create = "5m"
    delete = "5m"
  }
}
`
