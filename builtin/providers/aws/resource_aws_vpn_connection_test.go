package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVpnConnection_basic(t *testing.T) {
	rInt := acctest.RandInt()
	rBgpAsn := acctest.RandIntRange(64512, 65534)
	var vpn ec2.VpnConnection

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_connection.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccAwsVpnConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsVpnConnectionConfig(rBgpAsn),
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnection(
						"aws_vpc.vpc",
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.foo",
						&vpn,
					),
				),
			},
			{
				Config: testAccAwsVpnConnectionConfigUpdate(rInt, rBgpAsn),
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnection(
						"aws_vpc.vpc",
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.foo",
						&vpn,
					),
				),
			},
		},
	})
}

func TestAccAWSVpnConnection_withoutStaticRoutes(t *testing.T) {
	rInt := acctest.RandInt()
	rBgpAsn := acctest.RandIntRange(64512, 65534)
	var vpn ec2.VpnConnection
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_connection.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccAwsVpnConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsVpnConnectionConfigUpdate(rInt, rBgpAsn),
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnection(
						"aws_vpc.vpc",
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.foo",
						&vpn,
					),
					resource.TestCheckResourceAttr("aws_vpn_connection.foo", "static_routes_only", "false"),
				),
			},
		},
	})
}

func TestAccAWSVpnConnection_disappears(t *testing.T) {
	rBgpAsn := acctest.RandIntRange(64512, 65534)
	var vpn ec2.VpnConnection

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAwsVpnConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsVpnConnectionConfig(rBgpAsn),
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVpnConnection(
						"aws_vpc.vpc",
						"aws_vpn_gateway.vpn_gateway",
						"aws_customer_gateway.customer_gateway",
						"aws_vpn_connection.foo",
						&vpn,
					),
					testAccAWSVpnConnectionDisappears(&vpn),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAWSVpnConnectionDisappears(connection *ec2.VpnConnection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		_, err := conn.DeleteVpnConnection(&ec2.DeleteVpnConnectionInput{
			VpnConnectionId: connection.VpnConnectionId,
		})
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
				return nil
			}
			if err != nil {
				return err
			}
		}

		return resource.Retry(40*time.Minute, func() *resource.RetryError {
			opts := &ec2.DescribeVpnConnectionsInput{
				VpnConnectionIds: []*string{connection.VpnConnectionId},
			}
			resp, err := conn.DescribeVpnConnections(opts)
			if err != nil {
				cgw, ok := err.(awserr.Error)
				if ok && cgw.Code() == "InvalidVpnConnectionID.NotFound" {
					return nil
				}
				if ok && cgw.Code() == "IncorrectState" {
					return resource.RetryableError(fmt.Errorf(
						"Waiting for VPN Connection to be in the correct state: %v", connection.VpnConnectionId))
				}
				return resource.NonRetryableError(
					fmt.Errorf("Error retrieving VPN Connection: %s", err))
			}
			if *resp.VpnConnections[0].State == "deleted" {
				return nil
			}
			return resource.RetryableError(fmt.Errorf(
				"Waiting for VPN Connection: %v", connection.VpnConnectionId))
		})
	}
}

func testAccAwsVpnConnectionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpn_connection" {
			continue
		}

		resp, err := conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
			VpnConnectionIds: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpnConnectionID.NotFound" {
				// not found
				return nil
			}
			return err
		}

		var vpn *ec2.VpnConnection
		for _, v := range resp.VpnConnections {
			if v.VpnConnectionId != nil && *v.VpnConnectionId == rs.Primary.ID {
				vpn = v
			}
		}

		if vpn == nil {
			// vpn connection not found
			return nil
		}

		if vpn.State != nil && *vpn.State == "deleted" {
			return nil
		}

	}

	return nil
}

func testAccAwsVpnConnection(
	vpcResource string,
	vpnGatewayResource string,
	customerGatewayResource string,
	vpnConnectionResource string,
	vpnConnection *ec2.VpnConnection) resource.TestCheckFunc {
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

		resp, err := ec2conn.DescribeVpnConnections(&ec2.DescribeVpnConnectionsInput{
			VpnConnectionIds: []*string{aws.String(connection.Primary.ID)},
		})

		if err != nil {
			return err
		}

		*vpnConnection = *resp.VpnConnections[0]

		return nil
	}
}

func TestAWSVpnConnection_xmlconfig(t *testing.T) {
	tunnelInfo, err := xmlConfigToTunnelInfo(testAccAwsVpnTunnelInfoXML)
	if err != nil {
		t.Fatalf("Error unmarshalling XML: %s", err)
	}
	if tunnelInfo.Tunnel1Address != "FIRST_ADDRESS" {
		t.Fatalf("First address from tunnel XML was incorrect.")
	}
	if tunnelInfo.Tunnel1CgwInsideAddress != "FIRST_CGW_INSIDE_ADDRESS" {
		t.Fatalf("First Customer Gateway inside address from tunnel" +
			" XML was incorrect.")
	}
	if tunnelInfo.Tunnel1VgwInsideAddress != "FIRST_VGW_INSIDE_ADDRESS" {
		t.Fatalf("First VPN Gateway inside address from tunnel " +
			" XML was incorrect.")
	}
	if tunnelInfo.Tunnel1PreSharedKey != "FIRST_KEY" {
		t.Fatalf("First key from tunnel XML was incorrect.")
	}
	if tunnelInfo.Tunnel2Address != "SECOND_ADDRESS" {
		t.Fatalf("Second address from tunnel XML was incorrect.")
	}
	if tunnelInfo.Tunnel2CgwInsideAddress != "SECOND_CGW_INSIDE_ADDRESS" {
		t.Fatalf("Second Customer Gateway inside address from tunnel" +
			" XML was incorrect.")
	}
	if tunnelInfo.Tunnel2VgwInsideAddress != "SECOND_VGW_INSIDE_ADDRESS" {
		t.Fatalf("Second VPN Gateway inside address from tunnel " +
			" XML was incorrect.")
	}
	if tunnelInfo.Tunnel2PreSharedKey != "SECOND_KEY" {
		t.Fatalf("Second key from tunnel XML was incorrect.")
	}
}

func testAccAwsVpnConnectionConfig(rBgpAsn int) string {
	return fmt.Sprintf(`
		resource "aws_vpn_gateway" "vpn_gateway" {
		  tags {
		    Name = "vpn_gateway"
		  }
		}

		resource "aws_customer_gateway" "customer_gateway" {
		  bgp_asn = %d
		  ip_address = "178.0.0.1"
		  type = "ipsec.1"
			tags {
				Name = "main-customer-gateway"
			}
		}

		resource "aws_vpn_connection" "foo" {
		  vpn_gateway_id = "${aws_vpn_gateway.vpn_gateway.id}"
		  customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
		  type = "ipsec.1"
		  static_routes_only = true
		}
		`, rBgpAsn)
}

// Change static_routes_only to be false, forcing a refresh.
func testAccAwsVpnConnectionConfigUpdate(rInt, rBgpAsn int) string {
	return fmt.Sprintf(`
	resource "aws_vpn_gateway" "vpn_gateway" {
	  tags {
	    Name = "vpn_gateway"
	  }
	}

	resource "aws_customer_gateway" "customer_gateway" {
	  bgp_asn = %d
	  ip_address = "178.0.0.1"
	  type = "ipsec.1"
		tags {
	    Name = "main-customer-gateway-%d"
	  }
	}

	resource "aws_vpn_connection" "foo" {
	  vpn_gateway_id = "${aws_vpn_gateway.vpn_gateway.id}"
	  customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
	  type = "ipsec.1"
	  static_routes_only = false
	}
	`, rBgpAsn, rInt)
}

// Test our VPN tunnel config XML parsing
const testAccAwsVpnTunnelInfoXML = `
<vpn_connection id="vpn-abc123">
  <ipsec_tunnel>
    <customer_gateway>
      <tunnel_outside_address>
        <ip_address>123.123.123.123</ip_address>
      </tunnel_outside_address>
      <tunnel_inside_address>
        <ip_address>SECOND_CGW_INSIDE_ADDRESS</ip_address>
        <network_mask>255.255.255.252</network_mask>
        <network_cidr>30</network_cidr>
      </tunnel_inside_address>
    </customer_gateway>
    <vpn_gateway>
      <tunnel_outside_address>
        <ip_address>SECOND_ADDRESS</ip_address>
      </tunnel_outside_address>
      <tunnel_inside_address>
        <ip_address>SECOND_VGW_INSIDE_ADDRESS</ip_address>
        <network_mask>255.255.255.252</network_mask>
        <network_cidr>30</network_cidr>
      </tunnel_inside_address>
    </vpn_gateway>
    <ike>
      <pre_shared_key>SECOND_KEY</pre_shared_key>
    </ike>
  </ipsec_tunnel>
  <ipsec_tunnel>
    <customer_gateway>
      <tunnel_outside_address>
        <ip_address>123.123.123.123</ip_address>
      </tunnel_outside_address>
      <tunnel_inside_address>
        <ip_address>FIRST_CGW_INSIDE_ADDRESS</ip_address>
        <network_mask>255.255.255.252</network_mask>
        <network_cidr>30</network_cidr>
      </tunnel_inside_address>
    </customer_gateway>
    <vpn_gateway>
      <tunnel_outside_address>
        <ip_address>FIRST_ADDRESS</ip_address>
      </tunnel_outside_address>
      <tunnel_inside_address>
        <ip_address>FIRST_VGW_INSIDE_ADDRESS</ip_address>
        <network_mask>255.255.255.252</network_mask>
        <network_cidr>30</network_cidr>
      </tunnel_inside_address>
    </vpn_gateway>
    <ike>
      <pre_shared_key>FIRST_KEY</pre_shared_key>
    </ike>
  </ipsec_tunnel>
</vpn_connection>
`
