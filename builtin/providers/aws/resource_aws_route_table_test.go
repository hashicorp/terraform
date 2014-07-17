package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccAWSRouteTable_normal(t *testing.T) {
	var v ec2.RouteTable

	testCheck := func(*terraform.State) error {
		if len(v.Routes) != 2 {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}

		routes := make(map[string]ec2.Route)
		for _, r := range v.Routes {
			routes[r.DestinationCidrBlock] = r
		}

		if _, ok := routes["10.1.0.0/16"]; !ok {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}
		if _, ok := routes["10.2.0.0/16"]; !ok {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}

		return nil
	}

	testCheckChange := func(*terraform.State) error {
		if len(v.Routes) != 3 {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}

		routes := make(map[string]ec2.Route)
		for _, r := range v.Routes {
			routes[r.DestinationCidrBlock] = r
		}

		if _, ok := routes["10.1.0.0/16"]; !ok {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}
		if _, ok := routes["10.3.0.0/16"]; !ok {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}
		if _, ok := routes["10.4.0.0/16"]; !ok {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRouteTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRouteTableConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists(
						"aws_route_table.foo", &v),
					testCheck,
				),
			},

			resource.TestStep{
				Config: testAccRouteTableConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists(
						"aws_route_table.foo", &v),
					testCheckChange,
				),
			},
		},
	})
}

func TestAccAWSRouteTable_instance(t *testing.T) {
	var v ec2.RouteTable

	testCheck := func(*terraform.State) error {
		if len(v.Routes) != 2 {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}

		routes := make(map[string]ec2.Route)
		for _, r := range v.Routes {
			routes[r.DestinationCidrBlock] = r
		}

		if _, ok := routes["10.1.0.0/16"]; !ok {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}
		if _, ok := routes["10.2.0.0/16"]; !ok {
			return fmt.Errorf("bad routes: %#v", v.Routes)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRouteTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRouteTableConfigInstance,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableExists(
						"aws_route_table.foo", &v),
					testCheck,
				),
			},
		},
	})
}

func testAccCheckRouteTableDestroy(s *terraform.State) error {
	conn := testAccProvider.ec2conn

	for _, rs := range s.Resources {
		if rs.Type != "aws_route_table" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeRouteTables(
			[]string{rs.ID}, ec2.NewFilter())
		if err == nil {
			if len(resp.RouteTables) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		if ec2err.Code != "InvalidRouteTableID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckRouteTableExists(n string, v *ec2.RouteTable) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.ec2conn
		resp, err := conn.DescribeRouteTables(
			[]string{rs.ID}, ec2.NewFilter())
		if err != nil {
			return err
		}
		if len(resp.RouteTables) == 0 {
			return fmt.Errorf("RouteTable not found")
		}

		*v = resp.RouteTables[0]

		return nil
	}
}

const testAccRouteTableConfig = `
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
`

const testAccRouteTableConfigChange = `
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

const testAccRouteTableConfigInstance = `
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
