// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccAWSDefaultVpc_'
package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDefaultVpc_basic(t *testing.T) {
	var vpc ec2.Vpc

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultVpcDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDefaultVpcConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_default_vpc.foo", &vpc),
					testAccCheckVpcCidr(&vpc, "172.31.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_default_vpc.foo", "cidr_block", "172.31.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_default_vpc.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"aws_default_vpc.foo", "tags.Name", "Default VPC"),
					resource.TestCheckNoResourceAttr(
						"aws_default_vpc.foo", "assign_generated_ipv6_cidr_block"),
					resource.TestCheckNoResourceAttr(
						"aws_default_vpc.foo", "ipv6_association_id"),
					resource.TestCheckNoResourceAttr(
						"aws_default_vpc.foo", "ipv6_cidr_block"),
				),
			},
		},
	})
}

func testAccCheckAWSDefaultVpcDestroy(s *terraform.State) error {
	// We expect VPC to still exist
	return nil
}

const testAccAWSDefaultVpcConfigBasic = `
provider "aws" {
    region = "us-west-2"
}

resource "aws_default_vpc" "foo" {
	tags {
		Name = "Default VPC"
	}
}
`
