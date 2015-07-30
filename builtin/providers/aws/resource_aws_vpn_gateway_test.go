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

func TestAccAWSVpnGateway_basic(t *testing.T) {
	var v, v2 ec2.VPNGateway

	testNotEqual := func(*terraform.State) error {
		if len(v.VPCAttachments) == 0 {
			return fmt.Errorf("VPN gateway A is not attached")
		}
		if len(v2.VPCAttachments) == 0 {
			return fmt.Errorf("VPN gateway B is not attached")
		}

		id1 := v.VPCAttachments[0].VPCID
		id2 := v2.VPCAttachments[0].VPCID
		if id1 == id2 {
			return fmt.Errorf("Both attachment IDs are the same")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpnGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.foo", &v),
				),
			},

			resource.TestStep{
				Config: testAccVpnGatewayConfigChangeVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.foo", &v2),
					testNotEqual,
				),
			},
		},
	})
}

func TestAccAWSVpnGateway_delete(t *testing.T) {
	var vpnGateway ec2.VPNGateway

	testDeleted := func(r string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			_, ok := s.RootModule().Resources[r]
			if ok {
				return fmt.Errorf("VPN Gateway %q should have been deleted", r)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpnGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists("aws_vpn_gateway.foo", &vpnGateway)),
			},
			resource.TestStep{
				Config: testAccNoVpnGatewayConfig,
				Check:  resource.ComposeTestCheckFunc(testDeleted("aws_vpn_gateway.foo")),
			},
		},
	})
}

func TestAccAWSVpnGateway_tags(t *testing.T) {
	var v ec2.VPNGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVpnGatewayConfigTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists("aws_vpn_gateway.foo", &v),
					testAccCheckTags(&v.Tags, "foo", "bar"),
				),
			},

			resource.TestStep{
				Config: testAccCheckVpnGatewayConfigTagsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists("aws_vpn_gateway.foo", &v),
					testAccCheckTags(&v.Tags, "foo", ""),
					testAccCheckTags(&v.Tags, "bar", "baz"),
				),
			},
		},
	})
}

func testAccCheckVpnGatewayDestroy(s *terraform.State) error {
	ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpn_gateway" {
			continue
		}

		// Try to find the resource
		resp, err := ec2conn.DescribeVPNGateways(&ec2.DescribeVPNGatewaysInput{
			VPNGatewayIDs: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.VPNGateways) > 0 {
				return fmt.Errorf("still exists")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidVpnGatewayID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckVpnGatewayExists(n string, ig *ec2.VPNGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := ec2conn.DescribeVPNGateways(&ec2.DescribeVPNGatewaysInput{
			VPNGatewayIDs: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.VPNGateways) == 0 {
			return fmt.Errorf("VPNGateway not found")
		}

		*ig = *resp.VPNGateways[0]

		return nil
	}
}

const testAccNoVpnGatewayConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
`

const testAccVpnGatewayConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}
`

const testAccVpnGatewayConfigChangeVPC = `
resource "aws_vpc" "bar" {
	cidr_block = "10.2.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.bar.id}"
}
`

const testAccCheckVpnGatewayConfigTags = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		foo = "bar"
	}
}
`

const testAccCheckVpnGatewayConfigTagsUpdate = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		bar = "baz"
	}
}
`
