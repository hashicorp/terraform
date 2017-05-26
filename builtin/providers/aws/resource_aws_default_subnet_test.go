// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccAWSDefaultVpc_'
package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDefaultSubnet_basic(t *testing.T) {
	var v ec2.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDefaultSubnetConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists("aws_default_subnet.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_default_subnet.foo", "availability_zone", "us-west-2a"),
					resource.TestCheckResourceAttr(
						"aws_default_subnet.foo", "map_public_ip_on_launch", "true"),
					resource.TestCheckResourceAttr(
						"aws_default_subnet.foo", "assign_ipv6_address_on_creation", "false"),
					resource.TestCheckResourceAttr(
						"aws_default_subnet.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"aws_default_subnet.foo", "tags.Name", "Default subnet for us-west-2a"),
				),
			},
		},
	})
}

func testAccCheckAWSDefaultSubnetDestroy(s *terraform.State) error {
	// We expect subnet to still exist
	return nil
}

const testAccAWSDefaultSubnetConfigBasic = `
provider "aws" {
    region = "us-west-2"
}

resource "aws_default_subnet" "foo" {
	availability_zone = "us-west-2a"
	tags {
		Name = "Default subnet for us-west-2a"
	}
}
`
