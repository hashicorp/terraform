package alicloud

import (
	"fmt"
	"testing"

	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
)

func TestAccAlicloudSecurityGroup_basic(t *testing.T) {
	var sg ecs.DescribeSecurityGroupAttributeResponse

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_security_group.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityGroupExists(
						"alicloud_security_group.foo", &sg),
					resource.TestCheckResourceAttr(
						"alicloud_security_group.foo",
						"name",
						"sg_test"),
				),
			},
		},
	})

}

func TestAccAlicloudSecurityGroup_withVpc(t *testing.T) {
	var sg ecs.DescribeSecurityGroupAttributeResponse
	var vpc ecs.VpcSetType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_security_group.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSecurityGroupConfig_withVpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityGroupExists(
						"alicloud_security_group.foo", &sg),
					testAccCheckVpcExists(
						"alicloud_vpc.vpc", &vpc),
				),
			},
		},
	})

}

func testAccCheckSecurityGroupExists(n string, sg *ecs.DescribeSecurityGroupAttributeResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SecurityGroup ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		conn := client.ecsconn
		args := &ecs.DescribeSecurityGroupAttributeArgs{
			RegionId:        client.Region,
			SecurityGroupId: rs.Primary.ID,
		}
		d, err := conn.DescribeSecurityGroupAttribute(args)

		log.Printf("[WARN] security group id %#v", rs.Primary.ID)

		if err != nil {
			return err
		}

		if d == nil {
			return fmt.Errorf("SecurityGroup not found")
		}

		*sg = *d
		return nil
	}
}

func testAccCheckSecurityGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)
	conn := client.ecsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_security_group" {
			continue
		}

		// Try to find the SecurityGroup
		args := &ecs.DescribeSecurityGroupsArgs{
			RegionId: client.Region,
		}

		groups, _, err := conn.DescribeSecurityGroups(args)

		for _, sg := range groups {
			if sg.SecurityGroupId == rs.Primary.ID {
				return fmt.Errorf("Error SecurityGroup still exist")
			}
		}

		// Verify the error is what we want
		if err != nil {
			return err
		}
	}

	return nil
}

const testAccSecurityGroupConfig = `
resource "alicloud_security_group" "foo" {
  name = "sg_test"
}
`

const testAccSecurityGroupConfig_withVpc = `
resource "alicloud_security_group" "foo" {
  vpc_id = "${alicloud_vpc.vpc.id}"
}

resource "alicloud_vpc" "vpc" {
  cidr_block = "10.1.0.0/21"
}
`
