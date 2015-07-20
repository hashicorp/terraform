package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsVpnConnection_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAwsVpnConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsVpnConnectionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnection(
						"aws_vpc.vpc",
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsVpnConnectionConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnection(
						"aws_vpc.vpc",
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.foo",
					),
				),
			},
		},
	})
}

func testAccAwsVpnConnectionDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccAwsVpnConnection(
	vpcResource string,
	vpnGatewayResource string,
	customerGatewayResource string,
	vpnConnectionResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[vpnConnectionResource]
		if !ok {
			return fmt.Errorf("Not found: %s", vpnConnectionResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		connection, ok := s.RootModule().Resources[vpnConnectionResource]
		if !ok {
			return fmt.Errorf("Not found: %s", vpnConnectionResource)
		}

		ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn

		_, err := ec2conn.DescribeVPNConnections(&ec2.DescribeVPNConnectionsInput{
			VPNConnectionIDs: []*string{aws.String(connection.Primary.ID)},
		})

		if err != nil {
			return err
		}

		return nil
	}
}

const testAccAwsVpnConnectionConfig = `
resource "aws_vpn_gateway" "vpn_gateway" {
	tags {
		Name = "vpn_gateway"
	}
}

resource "aws_customer_gateway" "customer_gateway" {
	bgp_asn = 60000
	ip_address = "178.0.0.1"
	type = "ipsec.1"
}

resource "aws_vpn_connection" "foo" {
	vpn_gateway_id = "${aws_vpn_gateway.vpn_gateway.id}"
	customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
	type = "ipsec.1"
	static_routes_only = true
}
`

// Change static_routes_only to be false, forcing a refresh.
const testAccAwsVpnConnectionConfigUpdate = `
resource "aws_vpn_gateway" "vpn_gateway" {
	tags {
		Name = "vpn_gateway"
	}
}

resource "aws_customer_gateway" "customer_gateway" {
	bgp_asn = 60000
	ip_address = "178.0.0.1"
	type = "ipsec.1"
}

resource "aws_vpn_connection" "foo" {
	vpn_gateway_id = "${aws_vpn_gateway.vpn_gateway.id}"
	customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
	type = "ipsec.1"
	static_routes_only = false
}
`
