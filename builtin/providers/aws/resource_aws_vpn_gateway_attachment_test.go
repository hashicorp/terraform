package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVpnGatewayAttachment_basic(t *testing.T) {
	var vpc ec2.Vpc
	var vgw ec2.VpnGateway

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_gateway_attachment.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpnGatewayAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpnGatewayAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists(
						"aws_vpc.test",
						&vpc),
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.test",
						&vgw),
					testAccCheckVpnGatewayAttachmentExists(
						"aws_vpn_gateway_attachment.test",
						&vpc, &vgw),
				),
			},
		},
	})
}

func TestAccAWSVpnGatewayAttachment_deleted(t *testing.T) {
	var vpc ec2.Vpc
	var vgw ec2.VpnGateway

	testDeleted := func(n string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			_, ok := s.RootModule().Resources[n]
			if ok {
				return fmt.Errorf("Expected VPN Gateway attachment resource %q to be deleted.", n)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_gateway_attachment.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpnGatewayAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpnGatewayAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists(
						"aws_vpc.test",
						&vpc),
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.test",
						&vgw),
					testAccCheckVpnGatewayAttachmentExists(
						"aws_vpn_gateway_attachment.test",
						&vpc, &vgw),
				),
			},
			resource.TestStep{
				Config: testAccNoVpnGatewayAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testDeleted("aws_vpn_gateway_attachment.test"),
				),
			},
		},
	})
}

func testAccCheckVpnGatewayAttachmentExists(n string, vpc *ec2.Vpc, vgw *ec2.VpnGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		vpcId := rs.Primary.Attributes["vpc_id"]
		vgwId := rs.Primary.Attributes["vpn_gateway_id"]

		if len(vgw.VpcAttachments) == 0 {
			return fmt.Errorf("VPN Gateway %q has no attachments.", vgwId)
		}

		if *vgw.VpcAttachments[0].State != "attached" {
			return fmt.Errorf("Expected VPN Gateway %q to be in attached state, but got: %q",
				vgwId, *vgw.VpcAttachments[0].State)
		}

		if *vgw.VpcAttachments[0].VpcId != *vpc.VpcId {
			return fmt.Errorf("Expected VPN Gateway %q to be attached to VPC %q, but got: %q",
				vgwId, vpcId, *vgw.VpcAttachments[0].VpcId)
		}

		return nil
	}
}

func testAccCheckVpnGatewayAttachmentDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpn_gateway_attachment" {
			continue
		}

		vgwId := rs.Primary.Attributes["vpn_gateway_id"]

		resp, err := conn.DescribeVpnGateways(&ec2.DescribeVpnGatewaysInput{
			VpnGatewayIds: []*string{aws.String(vgwId)},
		})
		if err != nil {
			return err
		}

		vgw := resp.VpnGateways[0]
		if *vgw.VpcAttachments[0].State != "detached" {
			return fmt.Errorf("Expected VPN Gateway %q to be in detached state, but got: %q",
				vgwId, *vgw.VpcAttachments[0].State)
		}
	}

	return nil
}

const testAccNoVpnGatewayAttachmentConfig = `
resource "aws_vpc" "test" {
	cidr_block = "10.0.0.0/16"
}

resource "aws_vpn_gateway" "test" { }
`

const testAccVpnGatewayAttachmentConfig = `
resource "aws_vpc" "test" {
	cidr_block = "10.0.0.0/16"
}

resource "aws_vpn_gateway" "test" { }

resource "aws_vpn_gateway_attachment" "test" {
	vpc_id = "${aws_vpc.test.id}"
	vpn_gateway_id = "${aws_vpn_gateway.test.id}"
}
`
