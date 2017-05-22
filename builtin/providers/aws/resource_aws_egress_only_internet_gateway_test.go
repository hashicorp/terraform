package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEgressOnlyInternetGateway_basic(t *testing.T) {
	var igw ec2.EgressOnlyInternetGateway
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEgressOnlyInternetGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEgressOnlyInternetGatewayConfig_basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSEgressOnlyInternetGatewayExists("aws_egress_only_internet_gateway.foo", &igw),
				),
			},
		},
	})
}

func testAccCheckAWSEgressOnlyInternetGatewayDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_egress_only_internet_gateway" {
			continue
		}

		describe, err := conn.DescribeEgressOnlyInternetGateways(&ec2.DescribeEgressOnlyInternetGatewaysInput{
			EgressOnlyInternetGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})

		if err == nil {
			if len(describe.EgressOnlyInternetGateways) != 0 &&
				*describe.EgressOnlyInternetGateways[0].EgressOnlyInternetGatewayId == rs.Primary.ID {
				return fmt.Errorf("Egress Only Internet Gateway %q still exists", rs.Primary.ID)
			}
		}

		return nil
	}

	return nil
}

func testAccCheckAWSEgressOnlyInternetGatewayExists(n string, igw *ec2.EgressOnlyInternetGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Egress Only IGW ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeEgressOnlyInternetGateways(&ec2.DescribeEgressOnlyInternetGatewaysInput{
			EgressOnlyInternetGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.EgressOnlyInternetGateways) == 0 {
			return fmt.Errorf("Egress Only IGW not found")
		}

		*igw = *resp.EgressOnlyInternetGateways[0]

		return nil
	}
}

const testAccAWSEgressOnlyInternetGatewayConfig_basic = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	assign_generated_ipv6_cidr_block = true
	tags {
		Name = "testAccAWSEgressOnlyInternetGatewayConfig_basic"
	}
}

resource "aws_egress_only_internet_gateway" "foo" {
  	vpc_id = "${aws_vpc.foo.id}"
}
`
