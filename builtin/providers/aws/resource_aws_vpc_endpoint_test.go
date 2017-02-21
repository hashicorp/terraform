package aws

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVpcEndpoint_basic(t *testing.T) {
	var endpoint ec2.VpcEndpoint

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpc_endpoint.second-private-s3",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpcEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcEndpointWithRouteTableAndPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcEndpointExists("aws_vpc_endpoint.second-private-s3", &endpoint),
					testAccCheckVpcEndpointPrefixListAvailable("aws_vpc_endpoint.second-private-s3"),
				),
			},
		},
	})
}

func TestAccAWSVpcEndpoint_withRouteTableAndPolicy(t *testing.T) {
	var endpoint ec2.VpcEndpoint
	var routeTable ec2.RouteTable

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpc_endpoint.second-private-s3",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpcEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcEndpointWithRouteTableAndPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcEndpointExists("aws_vpc_endpoint.second-private-s3", &endpoint),
					testAccCheckRouteTableExists("aws_route_table.default", &routeTable),
				),
			},
			resource.TestStep{
				Config: testAccVpcEndpointWithRouteTableAndPolicyConfigModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcEndpointExists("aws_vpc_endpoint.second-private-s3", &endpoint),
					testAccCheckRouteTableExists("aws_route_table.default", &routeTable),
				),
			},
		},
	})
}

func TestAccAWSVpcEndpoint_WithoutRouteTableOrPolicyConfig(t *testing.T) {
	var endpoint ec2.VpcEndpoint

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpc_endpoint.second-private-s3",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpcEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcEndpointWithoutRouteTableOrPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcEndpointExists("aws_vpc_endpoint.second-private-s3", &endpoint),
					testAccCheckVpcEndpointPrefixListAvailable("aws_vpc_endpoint.second-private-s3"),
				),
			},
		},
	})
}

func TestAccAWSVpcEndpoint_removed(t *testing.T) {
	var endpoint ec2.VpcEndpoint

	// reach out and DELETE the VPC Endpoint outside of Terraform
	testDestroy := func(*terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		input := &ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: []*string{endpoint.VpcEndpointId},
		}

		_, err := conn.DeleteVpcEndpoints(input)
		if err != nil {
			return err
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcEndpointDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcEndpointWithoutRouteTableOrPolicyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcEndpointExists("aws_vpc_endpoint.second-private-s3", &endpoint),
					testDestroy,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckVpcEndpointDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_endpoint" {
			continue
		}

		// Try to find the VPC
		input := &ec2.DescribeVpcEndpointsInput{
			VpcEndpointIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeVpcEndpoints(input)
		if err != nil {
			// Verify the error is what we want
			if ae, ok := err.(awserr.Error); ok && ae.Code() == "InvalidVpcEndpointId.NotFound" {
				continue
			}
			return err
		}
		if len(resp.VpcEndpoints) > 0 {
			return fmt.Errorf("VPC Endpoints still exist.")
		}

		return err
	}

	return nil
}

func testAccCheckVpcEndpointExists(n string, endpoint *ec2.VpcEndpoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC Endpoint ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		input := &ec2.DescribeVpcEndpointsInput{
			VpcEndpointIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeVpcEndpoints(input)
		if err != nil {
			return err
		}
		if len(resp.VpcEndpoints) == 0 {
			return fmt.Errorf("VPC Endpoint not found")
		}

		*endpoint = *resp.VpcEndpoints[0]

		return nil
	}
}

func testAccCheckVpcEndpointPrefixListAvailable(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		prefixListID := rs.Primary.Attributes["prefix_list_id"]
		if prefixListID == "" {
			return fmt.Errorf("Prefix list ID not available")
		}
		if !strings.HasPrefix(prefixListID, "pl") {
			return fmt.Errorf("Prefix list ID does not appear to be a valid value: '%s'", prefixListID)
		}

		var (
			cidrBlockSize int
			err           error
		)

		if cidrBlockSize, err = strconv.Atoi(rs.Primary.Attributes["cidr_blocks.#"]); err != nil {
			return err
		}
		if cidrBlockSize < 1 {
			return fmt.Errorf("cidr_blocks seem suspiciously low: %d", cidrBlockSize)
		}

		return nil
	}
}

const testAccVpcEndpointWithRouteTableAndPolicyConfig = `
resource "aws_vpc" "foo" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "10.0.1.0/24"
}

resource "aws_vpc_endpoint" "second-private-s3" {
    vpc_id = "${aws_vpc.foo.id}"
    service_name = "com.amazonaws.us-west-2.s3"
    route_table_ids = ["${aws_route_table.default.id}"]
    policy = <<POLICY
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid":"AllowAll",
			"Effect":"Allow",
			"Principal":"*",
			"Action":"*",
			"Resource":"*"
		}
	]
}
POLICY
}

resource "aws_route_table" "default" {
    vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table_association" "main" {
    subnet_id = "${aws_subnet.foo.id}"
    route_table_id = "${aws_route_table.default.id}"
}
`

const testAccVpcEndpointWithRouteTableAndPolicyConfigModified = `
resource "aws_vpc" "foo" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "10.0.1.0/24"
}

resource "aws_vpc_endpoint" "second-private-s3" {
    vpc_id = "${aws_vpc.foo.id}"
    service_name = "com.amazonaws.us-west-2.s3"
    route_table_ids = ["${aws_route_table.default.id}"]
    policy = <<POLICY
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid":"AllowAll",
			"Effect":"Allow",
			"Principal":"*",
			"Action":"*",
			"Resource":"*"
		}
	]
}
POLICY
}

resource "aws_internet_gateway" "gw" {
    vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "default" {
    vpc_id = "${aws_vpc.foo.id}"

    route {
        cidr_block = "0.0.0.0/0"
        gateway_id = "${aws_internet_gateway.gw.id}"
    }
}

resource "aws_route_table_association" "main" {
    subnet_id = "${aws_subnet.foo.id}"
    route_table_id = "${aws_route_table.default.id}"
}
`

const testAccVpcEndpointWithoutRouteTableOrPolicyConfig = `
resource "aws_vpc" "foo" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_vpc_endpoint" "second-private-s3" {
    vpc_id = "${aws_vpc.foo.id}"
    service_name = "com.amazonaws.us-west-2.s3"
}
`
