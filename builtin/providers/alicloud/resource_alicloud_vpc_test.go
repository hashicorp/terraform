package alicloud

import (
	"fmt"
	"testing"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAlicloudVpc_basic(t *testing.T) {
	var vpc ecs.VpcSetType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_vpc.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("alicloud_vpc.foo", &vpc),
					resource.TestCheckResourceAttr(
						"alicloud_vpc.foo", "cidr_block", "172.16.0.0/12"),
					resource.TestCheckResourceAttrSet(
						"alicloud_vpc.foo", "router_id"),
					resource.TestCheckResourceAttrSet(
						"alicloud_vpc.foo", "router_table_id"),
				),
			},
		},
	})

}

func TestAccAlicloudVpc_update(t *testing.T) {
	var vpc ecs.VpcSetType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("alicloud_vpc.foo", &vpc),
					resource.TestCheckResourceAttr(
						"alicloud_vpc.foo", "cidr_block", "172.16.0.0/12"),
				),
			},
			resource.TestStep{
				Config: testAccVpcConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("alicloud_vpc.foo", &vpc),
					resource.TestCheckResourceAttr(
						"alicloud_vpc.foo", "name", "tf_test_bar"),
				),
			},
		},
	})
}

func testAccCheckVpcExists(n string, vpc *ecs.VpcSetType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		instance, err := client.DescribeVpc(rs.Primary.ID)

		if err != nil {
			return err
		}
		if instance == nil {
			return fmt.Errorf("VPC not found")
		}

		*vpc = *instance
		return nil
	}
}

func testAccCheckVpcDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_vpc" {
			continue
		}

		// Try to find the VPC
		instance, err := client.DescribeVpc(rs.Primary.ID)

		if instance != nil {
			return fmt.Errorf("VPCs still exist")
		}

		if err != nil {
			// Verify the error is what we want
			e, _ := err.(*common.Error)

			if e.ErrorResponse.Code != "InvalidVpcID.NotFound" {
				return err
			}
		}

	}

	return nil
}

const testAccVpcConfig = `
resource "alicloud_vpc" "foo" {
        name = "tf_test_foo"
        cidr_block = "172.16.0.0/12"
}
`

const testAccVpcConfigUpdate = `
resource "alicloud_vpc" "foo" {
	cidr_block = "172.16.0.0/12"
	name = "tf_test_bar"
}
`
