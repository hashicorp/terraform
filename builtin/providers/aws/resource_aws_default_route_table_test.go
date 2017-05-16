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
			{
				Config: testAccDefaultRouteTableConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists(
						"aws_default_route_table.foo", &v),
				),
			},
		},
	})
}

func TestAccAWSDefaultRouteTable_swap(t *testing.T) {
	var v ec2.RouteTable

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_default_route_table.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckDefaultRouteTableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDefaultRouteTable_change,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists(
						"aws_default_route_table.foo", &v),
				),
			},

			// This config will swap out the original Default Route Table and replace
			// it with the custom route table. While this is not advised, it's a
			// behavior that may happen, in which case a follow up plan will show (in
			// this case) a diff as the table now needs to be updated to match the
			// config
			{
				Config: testAccDefaultRouteTable_change_mod,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists(
						"aws_default_route_table.foo", &v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSDefaultRouteTable_vpc_endpoint(t *testing.T) {
	var v ec2.RouteTable

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_default_route_table.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckDefaultRouteTableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDefaultRouteTable_vpc_endpoint,
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

func testAccCheckDefaultRouteTableExists(s *terraform.State) error {
	// We can't destroy this resource; it comes and goes with the VPC itself.
	return nil
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

const testAccDefaultRouteTable_change = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "tf-default-route-table"
  }
}

resource "aws_default_route_table" "foo" {
  default_route_table_id = "${aws_vpc.foo.default_route_table_id}"

  route {
    cidr_block = "10.0.1.0/32"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }

  tags {
    Name = "this was the first main"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "main-igw"
  }
}

# Thing to help testing changes
resource "aws_route_table" "r" {
  vpc_id = "${aws_vpc.foo.id}"

  route {
    cidr_block = "10.0.1.0/24"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }

  tags {
    Name = "other"
  }
}
`

const testAccDefaultRouteTable_change_mod = `
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true

  tags {
    Name = "tf-default-route-table"
  }
}

resource "aws_default_route_table" "foo" {
  default_route_table_id = "${aws_vpc.foo.default_route_table_id}"

  route {
    cidr_block = "10.0.1.0/32"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }

  tags {
    Name = "this was the first main"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.foo.id}"

  tags {
    Name = "main-igw"
  }
}

# Thing to help testing changes
resource "aws_route_table" "r" {
  vpc_id = "${aws_vpc.foo.id}"

  route {
    cidr_block = "10.0.1.0/24"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }

  tags {
    Name = "other"
  }
}

resource "aws_main_route_table_association" "a" {
  vpc_id         = "${aws_vpc.foo.id}"
  route_table_id = "${aws_route_table.r.id}"
}
`

const testAccDefaultRouteTable_vpc_endpoint = `
provider "aws" {
    region = "us-west-2"
}

resource "aws_vpc" "test" {
    cidr_block = "10.0.0.0/16"

    tags {
        Name = "test"
    }
}

resource "aws_internet_gateway" "igw" {
    vpc_id = "${aws_vpc.test.id}"

    tags {
        Name = "test"
    }
}

resource "aws_vpc_endpoint" "s3" {
    vpc_id = "${aws_vpc.test.id}"
    service_name = "com.amazonaws.us-west-2.s3"
    route_table_ids = [
        "${aws_vpc.test.default_route_table_id}"
    ]
}

resource "aws_default_route_table" "foo" {
    default_route_table_id = "${aws_vpc.test.default_route_table_id}"

    tags {
        Name = "test"
    }

    route {
        cidr_block = "0.0.0.0/0"
        gateway_id = "${aws_internet_gateway.igw.id}"
    }
}
`
