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
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRouteBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSRoute_ipv6Support(t *testing.T) {
	var route ec2.Route

	//aws creates a default route
	testCheck := func(s *terraform.State) error {

		name := "aws_egress_only_internet_gateway.foo"
		gwres, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s\n", name)
		}

		if *route.EgressOnlyInternetGatewayId != gwres.Primary.ID {
			return fmt.Errorf("Egress Only Internet Gateway Id (Expected=%s, Actual=%s)\n", gwres.Primary.ID, *route.EgressOnlyInternetGatewayId)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRouteConfigIpv6,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testCheck,
				),
			},
		},
	})
}

func TestAccAWSRoute_ipv6ToInternetGateway(t *testing.T) {
	var route ec2.Route

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRouteConfigIpv6InternetGateway,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.igw", &route),
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
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRouteBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
					testCheck,
				),
			},
			{
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

func TestAccAWSRoute_noopdiff(t *testing.T) {
	var route ec2.Route
	var routeTable ec2.RouteTable

	testCheck := func(s *terraform.State) error {
		return nil
	}

	testCheckChange := func(s *terraform.State) error {
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRouteNoopChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.test", &route),
					testCheck,
				),
			},
			{
				Config: testAccAWSRouteNoopChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.test", &route),
					testAccCheckRouteTableExists("aws_route_table.test", &routeTable),
					testCheckChange,
				),
			},
		},
	})
}

func TestAccAWSRoute_doesNotCrashWithVPCEndpoint(t *testing.T) {
	var route ec2.Route

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRouteDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRouteWithVPCEndpoint,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRouteExists("aws_route.bar", &route),
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
			rs.Primary.Attributes["route_table_id"],
			rs.Primary.Attributes["destination_cidr_block"],
			rs.Primary.Attributes["destination_ipv6_cidr_block"],
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
			rs.Primary.Attributes["destination_ipv6_cidr_block"],
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

var testAccAWSRouteConfigIpv6InternetGateway = fmt.Sprintf(`
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
  assign_generated_ipv6_cidr_block = true
}

resource "aws_egress_only_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_internet_gateway" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "external" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route" "igw" {
  route_table_id = "${aws_route_table.external.id}"
  destination_ipv6_cidr_block = "::/0"
  gateway_id = "${aws_internet_gateway.foo.id}"
}

`)

var testAccAWSRouteConfigIpv6 = fmt.Sprintf(`
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
  assign_generated_ipv6_cidr_block = true
}

resource "aws_egress_only_internet_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route" "bar" {
	route_table_id = "${aws_route_table.foo.id}"
	destination_ipv6_cidr_block = "::/0"
	egress_only_gateway_id = "${aws_egress_only_internet_gateway.foo.id}"
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

var testAccAWSRouteNoopChange = fmt.Sprint(`
resource "aws_vpc" "test" {
  cidr_block = "10.10.0.0/16"
}

resource "aws_route_table" "test" {
  vpc_id = "${aws_vpc.test.id}"
}

resource "aws_subnet" "test" {
  vpc_id = "${aws_vpc.test.id}"
  cidr_block = "10.10.10.0/24"
}

resource "aws_route" "test" {
  route_table_id = "${aws_route_table.test.id}"
  destination_cidr_block = "0.0.0.0/0"
  instance_id = "${aws_instance.nat.id}"
}

resource "aws_instance" "nat" {
  ami = "ami-9abea4fb"
  instance_type = "t2.nano"
  subnet_id = "${aws_subnet.test.id}"
}
`)

var testAccAWSRouteWithVPCEndpoint = fmt.Sprint(`
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
  route_table_id         = "${aws_route_table.foo.id}"
  destination_cidr_block = "10.3.0.0/16"
  gateway_id             = "${aws_internet_gateway.foo.id}"

  # Forcing endpoint to create before route - without this the crash is a race.
  depends_on = ["aws_vpc_endpoint.baz"]
}

resource "aws_vpc_endpoint" "baz" {
  vpc_id          = "${aws_vpc.foo.id}"
  service_name    = "com.amazonaws.us-west-2.s3"
  route_table_ids = ["${aws_route_table.foo.id}"]
}
`)
