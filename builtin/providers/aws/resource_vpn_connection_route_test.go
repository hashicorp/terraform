package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsVpnConnectionRoute_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAwsVpnConnectionRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsVpnConnectionRouteConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnectionRoute(
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.vpn_connection",
						"aws_vpn_connection_route.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsVpnConnectionRouteConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnectionRoute(
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.vpn_connection",
						"aws_vpn_connection_route.foo",
					),
				),
			},
		},
	})
}

func testAccAwsVpnConnectionRouteDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccAwsVpnConnectionRoute(
	vpnGatewayResource string,
	customerGatewayResource string,
	vpnConnectionResource string,
	vpnConnectionRouteResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[vpnConnectionRouteResource]
		if !ok {
			return fmt.Errorf("Not found: %s", vpnConnectionRouteResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		route, ok := s.RootModule().Resources[vpnConnectionRouteResource]
		if !ok {
			return fmt.Errorf("Not found: %s", vpnConnectionRouteResource)
		}

		cidrBlock, vpnConnectionId := resourceAwsVpnConnectionRouteParseId(route.Primary.ID)

		routeFilters := []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("route.destination-cidr-block"),
				Values: []*string{aws.String(cidrBlock)},
			},
			&ec2.Filter{
				Name:   aws.String("vpn-connection-id"),
				Values: []*string{aws.String(vpnConnectionId)},
			},
		}

		ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn

		_, err := ec2conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
			Filters: routeFilters,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

const testAccAwsVpnConnectionRouteConfig = `
resource "aws_vpn_gateway" "vpn_gateway" {
	tags {
		Name = "vpn_gateway"
	}
}

resource "aws_customer_gateway" "customer_gateway" {
	bgp_asn = 60000
	ip_address = "182.0.0.1"
	type = "ipsec.1"
}

resource "aws_vpn_connection" "vpn_connection" {
	vpn_gateway_id = "${aws_vpn_gateway.vpn_gateway.id}"
	customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
	type = "ipsec.1"
	static_routes_only = true
}

resource "aws_vpn_connection_route" "foo" {
    destination_cidr_block = "172.168.10.0/24"
    vpn_connection_id = "${aws_vpn_connection.vpn_connection.id}"
}
`

// Change destination_cidr_block
const testAccAwsVpnConnectionRouteConfigUpdate = `
resource "aws_vpn_gateway" "vpn_gateway" {
	tags {
		Name = "vpn_gateway"
	}
}

resource "aws_customer_gateway" "customer_gateway" {
	bgp_asn = 60000
	ip_address = "182.0.0.1"
	type = "ipsec.1"
}

resource "aws_vpn_connection" "vpn_connection" {
	vpn_gateway_id = "${aws_vpn_gateway.vpn_gateway.id}"
	customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
	type = "ipsec.1"
	static_routes_only = true
}

resource "aws_vpn_connection_route" "foo" {
	destination_cidr_block = "172.168.20.0/24"
	vpn_connection_id = "${aws_vpn_connection.vpn_connection.id}"
}
`
