package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVPNGatewayRoutePropagation_basic(t *testing.T) {
	var rtID, gwID string

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_gateway_route_propagation.foo",
		Providers:     testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSVPNGatewayRoutePropagation_basic,
				Check: func(state *terraform.State) error {
					conn := testAccProvider.Meta().(*AWSClient).ec2conn

					rs := state.RootModule().Resources["aws_vpn_gateway_route_propagation.foo"]
					if rs == nil {
						return errors.New("missing resource state")
					}

					rtID = rs.Primary.Attributes["route_table_id"]
					gwID = rs.Primary.Attributes["vpn_gateway_id"]

					rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(conn, rtID)()
					if err != nil {
						return fmt.Errorf("failed to read route table: %s", err)
					}
					if rtRaw == nil {
						return errors.New("route table doesn't exist")
					}

					rt := rtRaw.(*ec2.RouteTable)
					exists := false
					for _, vgw := range rt.PropagatingVgws {
						if *vgw.GatewayId == gwID {
							exists = true
						}
					}
					if !exists {
						return errors.New("route table does not list VPN gateway as a propagator")
					}

					return nil
				},
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			conn := testAccProvider.Meta().(*AWSClient).ec2conn

			rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(conn, rtID)()
			if err != nil {
				return fmt.Errorf("failed to read route table: %s", err)
			}
			if rtRaw != nil {
				return errors.New("route table still exists")
			}
			return nil
		},
	})

}

const testAccAWSVPNGatewayRoutePropagation_basic = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route_table" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_vpn_gateway_route_propagation" "foo" {
	vpn_gateway_id = "${aws_vpn_gateway.foo.id}"
	route_table_id = "${aws_route_table.foo.id}"
}
`
