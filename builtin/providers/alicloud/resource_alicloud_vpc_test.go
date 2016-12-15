package alicloud

import (
	"fmt"
	"testing"

	"github.com/denverdino/aliyungo/ecs"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAliCloudVpc_basic(t *testing.T) {
	var vpc ecs.VpcSetType

	resource.Test(t, resource.TestCase{
		PreCheck:     func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("alicloud_vpc.foo", &vpc),
					testAccCheckVpcCidr(&vpc, "10.1.0.0/16"),
					resource.TestCheckResourceAttr(
						"alicloud_vpc.foo", "cidr_block", "10.1.0.0/16"),
				),
			},
		},
	})
}

const testAccVpcConfig = `
resource "alicloud_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
`

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
		resp, err := client.DescribeVpc(rs.Primary.ID)
		if err != nil {
			return err
		}

		if resp == nil {
			return fmt.Errorf("Not found: %s", n)
		}

		*vpc = *resp

		return nil
	}
}

func testAccCheckVpcCidr(vpc *ecs.VpcSetType, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if vpc.CidrBlock != expected {
			return fmt.Errorf("Bad cidr: %s", vpc.CidrBlock)
		}

		return nil
	}
}

func testAccCheckVpcDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_vpc" {
			continue
		}

		vpc, err := client.DescribeVpc(rs.Primary.ID)
		if err == nil && vpc != nil {
			return fmt.Errorf("VPCs still exist")
		}

		if err != nil {
			return err
		}

		return nil
	}

	return nil
}
