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

func TestAccAWSSubnet_basic(t *testing.T) {
	var v ec2.Subnet

	testCheck := func(*terraform.State) error {
		if *v.CidrBlock != "10.1.1.0/24" {
			return fmt.Errorf("bad cidr: %s", *v.CidrBlock)
		}

		if *v.MapPublicIpOnLaunch != true {
			return fmt.Errorf("bad MapPublicIpOnLaunch: %t", *v.MapPublicIpOnLaunch)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_subnet.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(
						"aws_subnet.foo", &v),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSSubnet_ipv6(t *testing.T) {
	var before, after ec2.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_subnet.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfigIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(
						"aws_subnet.foo", &before),
					testAccCheckAwsSubnetIpv6BeforeUpdate(t, &before),
				),
			},
			{
				Config: testAccSubnetConfigIpv6UpdateAssignIpv6OnCreation,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(
						"aws_subnet.foo", &after),
					testAccCheckAwsSubnetIpv6AfterUpdate(t, &after),
				),
			},
			{
				Config: testAccSubnetConfigIpv6UpdateIpv6Cidr,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(
						"aws_subnet.foo", &after),

					testAccCheckAwsSubnetNotRecreated(t, &before, &after),
				),
			},
		},
	})
}

func TestAccAWSSubnet_enableIpv6(t *testing.T) {
	var subnet ec2.Subnet

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_subnet.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckSubnetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfigPreIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(
						"aws_subnet.foo", &subnet),
				),
			},
			{
				Config: testAccSubnetConfigIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(
						"aws_subnet.foo", &subnet),
				),
			},
		},
	})
}

func testAccCheckAwsSubnetIpv6BeforeUpdate(t *testing.T, subnet *ec2.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if subnet.Ipv6CidrBlockAssociationSet == nil {
			return fmt.Errorf("Expected IPV6 CIDR Block Association")
		}

		if *subnet.AssignIpv6AddressOnCreation != true {
			return fmt.Errorf("bad AssignIpv6AddressOnCreation: %t", *subnet.AssignIpv6AddressOnCreation)
		}

		return nil
	}
}

func testAccCheckAwsSubnetIpv6AfterUpdate(t *testing.T, subnet *ec2.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *subnet.AssignIpv6AddressOnCreation != false {
			return fmt.Errorf("bad AssignIpv6AddressOnCreation: %t", *subnet.AssignIpv6AddressOnCreation)
		}

		return nil
	}
}

func testAccCheckAwsSubnetNotRecreated(t *testing.T,
	before, after *ec2.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.SubnetId != *after.SubnetId {
			t.Fatalf("Expected SubnetIDs not to change, but both got before: %s and after: %s", *before.SubnetId, *after.SubnetId)
		}
		return nil
	}
}

func testAccCheckSubnetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_subnet" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeSubnets(&ec2.DescribeSubnetsInput{
			SubnetIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.Subnets) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidSubnetID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckSubnetExists(n string, v *ec2.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeSubnets(&ec2.DescribeSubnetsInput{
			SubnetIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.Subnets) == 0 {
			return fmt.Errorf("Subnet not found")
		}

		*v = *resp.Subnets[0]

		return nil
	}
}

const testAccSubnetConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
	tags {
		Name = "tf-subnet-acc-test"
	}
}
`

const testAccSubnetConfigPreIpv6 = `
resource "aws_vpc" "foo" {
	cidr_block = "10.10.0.0/16"
	assign_generated_ipv6_cidr_block = true
}

resource "aws_subnet" "foo" {
	cidr_block = "10.10.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	map_public_ip_on_launch = true
	tags {
		Name = "tf-subnet-acc-test"
	}
}
`

const testAccSubnetConfigIpv6 = `
resource "aws_vpc" "foo" {
	cidr_block = "10.10.0.0/16"
	assign_generated_ipv6_cidr_block = true
}

resource "aws_subnet" "foo" {
	cidr_block = "10.10.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	ipv6_cidr_block = "${cidrsubnet(aws_vpc.foo.ipv6_cidr_block, 8, 1)}"
	map_public_ip_on_launch = true
	assign_ipv6_address_on_creation = true
	tags {
		Name = "tf-subnet-acc-test"
	}
}
`

const testAccSubnetConfigIpv6UpdateAssignIpv6OnCreation = `
resource "aws_vpc" "foo" {
	cidr_block = "10.10.0.0/16"
	assign_generated_ipv6_cidr_block = true
}

resource "aws_subnet" "foo" {
	cidr_block = "10.10.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	ipv6_cidr_block = "${cidrsubnet(aws_vpc.foo.ipv6_cidr_block, 8, 1)}"
	map_public_ip_on_launch = true
	assign_ipv6_address_on_creation = false
	tags {
		Name = "tf-subnet-acc-test"
	}
}
`

const testAccSubnetConfigIpv6UpdateIpv6Cidr = `
resource "aws_vpc" "foo" {
	cidr_block = "10.10.0.0/16"
	assign_generated_ipv6_cidr_block = true
}

resource "aws_subnet" "foo" {
	cidr_block = "10.10.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
	ipv6_cidr_block = "${cidrsubnet(aws_vpc.foo.ipv6_cidr_block, 8, 3)}"
	map_public_ip_on_launch = true
	assign_ipv6_address_on_creation = false
	tags {
		Name = "tf-subnet-acc-test"
	}
}
`
