package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccAlicloudNatGateway_basic(t *testing.T) {
	var nat ecs.NatGatewaySetType

	testCheck := func(*terraform.State) error {
		if nat.BusinessStatus != "Normal" {
			return fmt.Errorf("abnormal instance status")
		}

		if len(nat.BandwidthPackageIds.BandwidthPackageId) == 0 {
			return fmt.Errorf("no bandwidth package: %#v", nat.BandwidthPackageIds.BandwidthPackageId)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_nat_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckNatGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNatGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNatGatewayExists(
						"alicloud_nat_gateway.foo", &nat),
					testCheck,
					resource.TestCheckResourceAttr(
						"alicloud_nat_gateway.foo",
						"spec",
						"Small"),
					resource.TestCheckResourceAttr(
						"alicloud_nat_gateway.foo",
						"name",
						"test_foo"),
					testAccCheckNatgatewayIpAddress("alicloud_nat_gateway.foo", &nat),
				),
			},
		},
	})

}

func TestAccAlicloudNatGateway_spec(t *testing.T) {
	var nat ecs.NatGatewaySetType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_nat_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckNatGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNatGatewayConfigSpec,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNatGatewayExists(
						"alicloud_nat_gateway.foo", &nat),
					resource.TestCheckResourceAttr(
						"alicloud_nat_gateway.foo",
						"spec",
						"Middle"),
				),
			},

			resource.TestStep{
				Config: testAccNatGatewayConfigSpecUpgrade,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNatGatewayExists(
						"alicloud_nat_gateway.foo", &nat),
					resource.TestCheckResourceAttr(
						"alicloud_nat_gateway.foo",
						"spec",
						"Large"),
				),
			},
		},
	})

}

func testAccCheckNatgatewayIpAddress(n string, nat *ecs.NatGatewaySetType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No NatGateway ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		natGateway, err := client.DescribeNatGateway(rs.Primary.ID)

		if err != nil {
			return err
		}
		if natGateway == nil {
			return fmt.Errorf("Natgateway not found")
		}

		return nil
	}
}

func testAccCheckNatGatewayExists(n string, nat *ecs.NatGatewaySetType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Gateway ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		instance, err := client.DescribeNatGateway(rs.Primary.ID)

		if err != nil {
			return err
		}
		if instance == nil {
			return fmt.Errorf("Nat gateway not found")
		}

		*nat = *instance
		return nil
	}
}

func testAccCheckNatGatewayDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_nat_gateway" {
			continue
		}

		// Try to find the Nat gateway
		instance, err := client.DescribeNatGateway(rs.Primary.ID)

		if instance != nil {
			return fmt.Errorf("Nat gateway still exist")
		}

		if err != nil {
			// Verify the error is what we want
			e, _ := err.(*common.Error)

			if !notFoundError(e) {
				return err
			}
		}

	}

	return nil
}

const testAccNatGatewayConfig = `
data "alicloud_zones" "default" {
	"available_resource_creation"= "VSwitch"
}

resource "alicloud_vpc" "foo" {
	name = "tf_test_foo"
	cidr_block = "172.16.0.0/12"
}

resource "alicloud_vswitch" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	cidr_block = "172.16.0.0/21"
	availability_zone = "${data.alicloud_zones.default.zones.2.id}"
}

resource "alicloud_nat_gateway" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	spec = "Small"
	name = "test_foo"
	bandwidth_packages = [{
	  ip_count = 1
	  bandwidth = 5
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	}, {
	  ip_count = 2
	  bandwidth = 6
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	}, {
	  ip_count = 3
	  bandwidth = 7
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	}, {
	  ip_count = 1
	  bandwidth = 8
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	}]
	depends_on = [
    	"alicloud_vswitch.foo"]
}
`

const testAccNatGatewayConfigSpec = `
data "alicloud_zones" "default" {
	"available_resource_creation"= "VSwitch"
}

resource "alicloud_vpc" "foo" {
	name = "tf_test_foo"
	cidr_block = "172.16.0.0/12"
}

resource "alicloud_vswitch" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	cidr_block = "172.16.0.0/21"
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
}

resource "alicloud_nat_gateway" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	spec = "Middle"
	name = "test_foo"
	bandwidth_packages = [{
	  ip_count = 1
	  bandwidth = 5
	  zone = "${data.alicloud_zones.default.zones.0.id}"
	}, {
	  ip_count = 2
	  bandwidth = 10
	  zone = "${data.alicloud_zones.default.zones.0.id}"
	}]
	depends_on = [
    	"alicloud_vswitch.foo"]
}
`

const testAccNatGatewayConfigSpecUpgrade = `
data "alicloud_zones" "default" {
	"available_resource_creation"= "VSwitch"
}

resource "alicloud_vpc" "foo" {
	name = "tf_test_foo"
	cidr_block = "172.16.0.0/12"
}

resource "alicloud_vswitch" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	cidr_block = "172.16.0.0/21"
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
}

resource "alicloud_nat_gateway" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	spec = "Large"
	name = "test_foo"
	bandwidth_packages = [{
	  ip_count = 1
	  bandwidth = 5
	  zone = "${data.alicloud_zones.default.zones.0.id}"
	}, {
	  ip_count = 2
	  bandwidth = 10
	  zone = "${data.alicloud_zones.default.zones.0.id}"
	}]
	depends_on = [
    	"alicloud_vswitch.foo"]
}
`
