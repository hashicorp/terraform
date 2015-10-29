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

	//aws creates a default route
	testCheck := func(s *terraform.State) error {
		if *route.DestinationCidrBlock != "10.3.0.0/16" {
			return fmt.Errorf("Destination Cidr (Expected=%s, Actual=%s)\n", "10.3.0.0/16", *route.DestinationCidrBlock)
		}

		name := "aws_internet_gateway.foo"
		gwres, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s\n", name)
		}

		if *route.GatewayId != gwres.Primary.ID {
			return fmt.Errorf("Internet Gateway Id (Expected=%s, Actual=%s)\n", gwres.Primary.ID, *route.GatewayId)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRouteBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSRoute_changeCidr(t *testing.T) {
	var route ec2.Route
	var routeTable ec2.RouteTable

	//aws creates a default route
	testCheck := func(s *terraform.State) error {
		if *route.DestinationCidrBlock != "10.3.0.0/16" {
			return fmt.Errorf("Destination Cidr (Expected=%s, Actual=%s)\n", "10.3.0.0/16", *route.DestinationCidrBlock)
		}

		name := "aws_internet_gateway.foo"
		gwres, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s\n", name)
		}

		if *route.GatewayId != gwres.Primary.ID {
			return fmt.Errorf("Internet Gateway Id (Expected=%s, Actual=%s)\n", gwres.Primary.ID, *route.GatewayId)
		}

		return nil
	}

	testCheckChange := func(s *terraform.State) error {
		if *route.DestinationCidrBlock != "10.2.0.0/16" {
			return fmt.Errorf("Destination Cidr (Expected=%s, Actual=%s)\n", "10.2.0.0/16", *route.DestinationCidrBlock)
		}

		name := "aws_internet_gateway.foo"
		gwres, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s\n", name)
		}

		if *route.GatewayId != gwres.Primary.ID {
			return fmt.Errorf("Internet Gateway Id (Expected=%s, Actual=%s)\n", gwres.Primary.ID, *route.GatewayId)
		}

		if rtlen := len(routeTable.Routes); rtlen != 2 {
			return fmt.Errorf("Route Table has too many routes (Expected=%d, Actual=%d)\n", rtlen, 2)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRouteBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testCheck,
				),
			},
			resource.TestStep{
				Config: testAccAWSRouteBasicConfigChangeCidr,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testAccCheckRouteTableExists("aws_route_table.foo", &routeTable),
					testCheckChange,
				),
			},
		},
	})
}

// Acceptance test if mixed inline and external routes are implemented
/*
func TestAccAWSRoute_mix(t *testing.T) {
	var rt ec2.RouteTable
	var route ec2.Route

	//aws creates a default route
	testCheck := func(s *terraform.State) error {
		if *route.DestinationCidrBlock != "0.0.0.0/0" {
			return fmt.Errorf("Destination Cidr (Expected=%s, Actual=%s)\n", "0.0.0.0/0", *route.DestinationCidrBlock)
		}

		name := "aws_internet_gateway.foo"
		gwres, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s\n", name)
		}

		if *route.GatewayId != gwres.Primary.ID {
			return fmt.Errorf("Internet Gateway Id (Expected=%s, Actual=%s)\n", gwres.Primary.ID, *route.GatewayId)
		}

		if len(rt.Routes) != 3 {
			return fmt.Errorf("bad routes: %#v", rt.Routes)
		}

		routes := make(map[string]*ec2.Route)
		for _, r := range rt.Routes {
			routes[*r.DestinationCidrBlock] = r
		}

		if _, ok := routes["10.1.0.0/16"]; !ok {
			return fmt.Errorf("Missing route %s: %#v", "10.1.0.0/16", rt.Routes)
		}
		if _, ok := routes["10.2.0.0/16"]; !ok {
			return fmt.Errorf("Missing route %s: %#v", "10.2.0.0/16", rt.Routes)
		}
		if _, ok := routes["0.0.0.0/0"]; !ok {
			return fmt.Errorf("Missing route %s: %#v", "0.0.0.0/0", rt.Routes)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRouteMixConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists("aws_route_table.foo", &rt),
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testCheck,
				),
			},
		},
	})
}
*/

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
			rs.Primary.Attributes["route_table_id"],
			rs.Primary.Attributes["destination_cidr_block"],
		)

		if err != nil {
			return err
		}

		if r == nil {
			return fmt.Errorf("Route not found")
		}

		*res = *r

		return nil
	}
}

func testAccCheckAWSRouteDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		route, err := findResourceRoute(
			conn,
			rs.Primary.Attributes["route_table_id"],
			rs.Primary.Attributes["destination_cidr_block"],
		)

		if route == nil && err == nil {
			return nil
		}
	}

	return nil
}

var testAccAWSRouteBasicConfig = fmt.Sprint(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route" "bar" {
	route_table_id = "${aws_route_table.foo.id}"
	destination_cidr_block = "10.3.0.0/16"
	gateway_id = "${aws_internet_gateway.foo.id}"
}
`)

var testAccAWSRouteBasicConfigChangeCidr = fmt.Sprint(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route" "bar" {
	route_table_id = "${aws_route_table.foo.id}"
	destination_cidr_block = "10.2.0.0/16"
	gateway_id = "${aws_internet_gateway.foo.id}"
}
`)

// Acceptance test if mixed inline and external routes are implemented
var testAccAWSRouteMixConfig = fmt.Sprint(`
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

resource "aws_route" "bar" {
	route_table_id = "${aws_route_table.foo.id}"
	destination_cidr_block = "0.0.0.0/0"
	gateway_id = "${aws_internet_gateway.foo.id}"
}
`)
