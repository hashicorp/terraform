package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRoute_basic(t *testing.T) {
	var route ec2.Route

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRouteConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testAccCheckAWSRouteAttributes(&route),
				),
			},
		},
	})
}

func testAccCheckAWSRouteExists(n string, res *ec2.Route) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s\n", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		r, err := findResourceRoute(
			conn,
			rs.Primary.ID,
			rs.Primary.Attributes["destination_cidr_block"],
		)

		if err != nil {
			return err
		}

		if r == nil {
			return fmt.Errorf("Route not found")
		}

		return nil
	}
}

func testAccCheckAWSRouteAttributes(route *ec2.Route) resource.TestCheckFunc {
	return func(s *terraform.State) {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		return nil
	}
}

func testAccCheckAWSRouteDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		route, err := findResourceRoute(
			conn,
			rs.Primary.ID,
			rs.Primary.Attributes["destination_cidr_block"],
		)

		if route == nil && err == nil {
			return nil
		}
	}

	return nil
}

var testAccAWSRouteConfig = fmt.Sprint(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	route {
		cidr_block = "10.2.0.0/16"
		gateway_id = "${aws_internet_gateway.foo.id}"
	}
}

resource "aws_route" "foo" {
	route_table_id = "${aws_route_table.foo.id}"
	destination_cidr_block = "${aws_vpc.foo.cidr_block}"
	gateway_id = "${aws_internet_gateway.foo.id}"
}
`)
