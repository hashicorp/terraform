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

func TestAccAWSRouteTableAssociation_basic(t *testing.T) {
	var v, v2 ec2.RouteTable

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRouteTableAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRouteTableAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableAssociationExists(
						"aws_route_table_association.foo", &v),
				),
			},

			resource.TestStep{
				Config: testAccRouteTableAssociationConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableAssociationExists(
						"aws_route_table_association.foo", &v2),
				),
			},
		},
	})
}

func testAccCheckRouteTableAssociationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route_table_association" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
			RouteTableIds: []*string{aws.String(rs.Primary.Attributes["route_table_id"])},
		})
		if err != nil {
			// Verify the error is what we want
			ec2err, ok := err.(awserr.Error)
			if !ok {
				return err
			}
			if ec2err.Code() != "InvalidRouteTableID.NotFound" {
				return err
			}
			return nil
		}

		rt := resp.RouteTables[0]
		if len(rt.Associations) > 0 {
			return fmt.Errorf(
				"route table %s has associations", *rt.RouteTableId)

		}
	}

	return nil
}

func testAccCheckRouteTableAssociationExists(n string, v *ec2.RouteTable) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
			RouteTableIds: []*string{aws.String(rs.Primary.Attributes["route_table_id"])},
		})
		if err != nil {
			return err
		}
		if len(resp.RouteTables) == 0 {
			return fmt.Errorf("RouteTable not found")
		}

		*v = *resp.RouteTables[0]

		if len(v.Associations) == 0 {
			return fmt.Errorf("no associations")
		}

		return nil
	}
}

const testAccRouteTableAssociationConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	cidr_block = "10.1.1.0/24"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	route {
		cidr_block = "10.0.0.0/8"
		gateway_id = "${aws_internet_gateway.foo.id}"
	}
}

resource "aws_route_table_association" "foo" {
	route_table_id = "${aws_route_table.foo.id}"
	subnet_id = "${aws_subnet.foo.id}"
}
`

const testAccRouteTableAssociationConfigChange = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	cidr_block = "10.1.1.0/24"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
	route {
		cidr_block = "10.0.0.0/8"
		gateway_id = "${aws_internet_gateway.foo.id}"
	}
}

resource "aws_route_table_association" "foo" {
	route_table_id = "${aws_route_table.bar.id}"
	subnet_id = "${aws_subnet.foo.id}"
}
`
