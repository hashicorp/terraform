package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSNatGateway_basic(t *testing.T) {
	var natGateway ec2.NatGateway

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_nat_gateway.gateway",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckNatGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNatGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNatGatewayExists("aws_nat_gateway.gateway", &natGateway),
				),
			},
		},
	})
}

func testAccCheckNatGatewayDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_nat_gateway" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.NatGateways) > 0 && strings.ToLower(*resp.NatGateways[0].State) != "deleted" {
				return fmt.Errorf("still exists")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "NatGatewayNotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckNatGatewayExists(n string, ng *ec2.NatGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.NatGateways) == 0 {
			return fmt.Errorf("NatGateway not found")
		}

		*ng = *resp.NatGateways[0]

		return nil
	}
}

const testAccNatGatewayConfig = `
resource "aws_vpc" "vpc" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "private" {
    vpc_id = "${aws_vpc.vpc.id}"
    cidr_block = "10.0.1.0/24"
    map_public_ip_on_launch = false
}

resource "aws_subnet" "public" {
    vpc_id = "${aws_vpc.vpc.id}"
    cidr_block = "10.0.2.0/24"
    map_public_ip_on_launch = true
}

resource "aws_internet_gateway" "gw" {
    vpc_id = "${aws_vpc.vpc.id}"
}

resource "aws_eip" "nat_gateway" {
    vpc = true
}

// Actual SUT
resource "aws_nat_gateway" "gateway" {
    allocation_id = "${aws_eip.nat_gateway.id}"
    subnet_id = "${aws_subnet.public.id}"

    depends_on = ["aws_internet_gateway.gw"]
}

resource "aws_route_table" "private" {
    vpc_id = "${aws_vpc.vpc.id}"

    route {
        cidr_block = "0.0.0.0/0"
        nat_gateway_id = "${aws_nat_gateway.gateway.id}"
    }
}

resource "aws_route_table_association" "private" {
    subnet_id = "${aws_subnet.private.id}"
    route_table_id = "${aws_route_table.private.id}"
}

resource "aws_route_table" "public" {
    vpc_id = "${aws_vpc.vpc.id}"

    route {
        cidr_block = "0.0.0.0/0"
        gateway_id = "${aws_internet_gateway.gw.id}"
    }
}

resource "aws_route_table_association" "public" {
    subnet_id = "${aws_subnet.public.id}"
    route_table_id = "${aws_route_table.public.id}"
}
`
