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

func TestAccAWSDefaultRouteTable_basic(t *testing.T) {
	var v ec2.RouteTable

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_default_route_table.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckDefaultRouteTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDefaultRouteTableConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists(
						"aws_default_route_table.foo", &v),
				),
			},
		},
	})
}

func testAccCheckDefaultRouteTableDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_default_route_table" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
			RouteTableIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.RouteTables) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidRouteTableID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckDefaultRouteTableExists(n string, v *ec2.RouteTable) resource.TestCheckFunc {
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
			RouteTableIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.RouteTables) == 0 {
			return fmt.Errorf("RouteTable not found")
		}

		*v = *resp.RouteTables[0]

		return nil
	}
}

const testAccDefaultRouteTableConfig = `
resource "aws_vpc" "foo" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "tf-default-route-table-test"
  }
}

resource "aws_default_route_table" "foo" {
  default_route_table_id = "${aws_vpc.foo.default_route_table_id}"

  route {
    cidr_block = "10.0.1.0/32"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }

  tags {
    Name = "tf-default-route-table-test"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-default-route-table-test"
  }
}`

const testAccDefaultRouteTableConfigChange = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"

	route {
		cidr_block = "10.3.0.0/16"
		gateway_id = "${aws_internet_gateway.foo.id}"
	}

	route {
		cidr_block = "10.4.0.0/16"
		gateway_id = "${aws_internet_gateway.foo.id}"
	}
}
`

const testAccDefaultRouteTableConfigInstance = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	subnet_id = "${aws_subnet.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"

	route {
		cidr_block = "10.2.0.0/16"
		instance_id = "${aws_instance.foo.id}"
	}
}
`

const testAccDefaultRouteTableConfigTags = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"

	tags {
		foo = "bar"
	}
}
`

const testAccDefaultRouteTableConfigTagsUpdate = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"

	tags {
		bar = "baz"
	}
}
`

// VPC Peering connections are prefixed with pcx
// This test requires an ENV var, AWS_ACCOUNT_ID, with a valid AWS Account ID
func testAccDefaultRouteTableVpcPeeringConfig(acc string) string {
	cfg := `resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_vpc" "bar" {
	cidr_block = "10.3.0.0/16"
}

resource "aws_internet_gateway" "bar" {
	vpc_id = "${aws_vpc.bar.id}"
}

resource "aws_vpc_peering_connection" "foo" {
		vpc_id = "${aws_vpc.foo.id}"
		peer_vpc_id = "${aws_vpc.bar.id}"
		peer_owner_id = "%s"
		tags {
			foo = "bar"
		}
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"

	route {
		cidr_block = "10.2.0.0/16"
		vpc_peering_connection_id = "${aws_vpc_peering_connection.foo.id}"
	}
}
`
	return fmt.Sprintf(cfg, acc)
}

const testAccDefaultRouteTableVgwRoutePropagationConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"

	propagating_vgws = ["${aws_vpn_gateway.foo.id}"]
}
`
