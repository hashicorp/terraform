package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVpcEndpointRouteTableAssociation_basic(t *testing.T) {
	var vpce ec2.VpcEndpoint

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcEndpointRouteTableAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcEndpointRouteTableAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcEndpointRouteTableAssociationExists(
						"aws_vpc_endpoint_route_table_association.a", &vpce),
				),
			},
		},
	})
}

func testAccCheckVpcEndpointRouteTableAssociationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_endpoint_route_table_association" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeVpcEndpoints(&ec2.DescribeVpcEndpointsInput{
			VpcEndpointIds: aws.StringSlice([]string{rs.Primary.Attributes["vpc_endpoint_id"]}),
		})
		if err != nil {
			// Verify the error is what we want
			ec2err, ok := err.(awserr.Error)
			if !ok {
				return err
			}
			if ec2err.Code() != "InvalidVpcEndpointId.NotFound" {
				return err
			}
			return nil
		}

		vpce := resp.VpcEndpoints[0]
		if len(vpce.RouteTableIds) > 0 {
			return fmt.Errorf(
				"VPC endpoint %s has route tables", *vpce.VpcEndpointId)
		}
	}

	return nil
}

func testAccCheckVpcEndpointRouteTableAssociationExists(n string, vpce *ec2.VpcEndpoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeVpcEndpoints(&ec2.DescribeVpcEndpointsInput{
			VpcEndpointIds: aws.StringSlice([]string{rs.Primary.Attributes["vpc_endpoint_id"]}),
		})
		if err != nil {
			return err
		}
		if len(resp.VpcEndpoints) == 0 {
			return fmt.Errorf("VPC endpoint not found")
		}

		*vpce = *resp.VpcEndpoints[0]

		if len(vpce.RouteTableIds) == 0 {
			return fmt.Errorf("no route table associations")
		}

		for _, id := range vpce.RouteTableIds {
			if *id == rs.Primary.Attributes["route_table_id"] {
				return nil
			}
		}

		return fmt.Errorf("route table association not found")
	}
}

const testAccVpcEndpointRouteTableAssociationConfig = `
provider "aws" {
    region = "us-west-2"
}

resource "aws_vpc" "foo" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_vpc_endpoint" "s3" {
    vpc_id = "${aws_vpc.foo.id}"
    service_name = "com.amazonaws.us-west-2.s3"
}

resource "aws_route_table" "rt" {
    vpc_id = "${aws_vpc.foo.id}"

    tags {
        Name = "test"
    }
}

resource "aws_vpc_endpoint_route_table_association" "a" {
	vpc_endpoint_id = "${aws_vpc_endpoint.s3.id}"
	route_table_id  = "${aws_route_table.rt.id}"
}
`
