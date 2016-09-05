package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVpnGateway_basic(t *testing.T) {
	var v, v2 ec2.VpnGateway

	testNotEqual := func(*terraform.State) error {
		if len(v.VpcAttachments) == 0 {
			return fmt.Errorf("VPN Gateway A is not attached")
		}
		if len(v2.VpcAttachments) == 0 {
			return fmt.Errorf("VPN Gateway B is not attached")
		}

		id1 := v.VpcAttachments[0].VpcId
		id2 := v2.VpcAttachments[0].VpcId
		if id1 == id2 {
			return fmt.Errorf("Both attachment IDs are the same")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpnGatewayDestroy,
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

func TestAccAWSVpnGateway_withAvailabilityZoneSetToState(t *testing.T) {
	var v ec2.VpnGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpnGatewayConfigWithAZ,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists("aws_vpn_gateway.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_vpn_gateway.foo", "availability_zone", "us-west-2a"),
				),
			},
		},
	})
}

func TestAccAWSVpnGateway_disappears(t *testing.T) {
	var v ec2.VpnGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpnGatewayConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists("aws_vpn_gateway.foo", &v),
					testAccAWSVpnGatewayDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSVpnGateway_reattach(t *testing.T) {
	var vpc1, vpc2 ec2.Vpc
	var vgw1, vgw2 ec2.VpnGateway

	testAttachmentFunc := func(vgw *ec2.VpnGateway, vpc *ec2.Vpc) func(*terraform.State) error {
		return func(*terraform.State) error {
			if len(vgw.VpcAttachments) == 0 {
				return fmt.Errorf("VPN Gateway %q has no VPC attachments.",
					*vgw.VpnGatewayId)
			}

			if len(vgw.VpcAttachments) > 1 {
				count := 0
				for _, v := range vgw.VpcAttachments {
					if *v.State == "attached" {
						count += 1
					}
				}
				if count > 1 {
					return fmt.Errorf(
						"VPN Gateway %q has an unexpected number of VPC attachments (more than 1): %#v",
						*vgw.VpnGatewayId, vgw.VpcAttachments)
				}
			}

			if *vgw.VpcAttachments[0].State != "attached" {
				return fmt.Errorf("Expected VPN Gateway %q to be attached.",
					*vgw.VpnGatewayId)
			}

			if *vgw.VpcAttachments[0].VpcId != *vpc.VpcId {
				return fmt.Errorf("Expected VPN Gateway %q to be attached to VPC %q, but got: %q",
					*vgw.VpnGatewayId, *vpc.VpcId, *vgw.VpcAttachments[0].VpcId)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVpnGatewayConfigReattach,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc1),
					testAccCheckVpcExists("aws_vpc.bar", &vpc2),
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.foo", &vgw1),
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.bar", &vgw2),
					testAttachmentFunc(&vgw1, &vpc1),
					testAttachmentFunc(&vgw2, &vpc2),
				),
			},
			resource.TestStep{
				Config: testAccCheckVpnGatewayConfigReattachChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.foo", &vgw1),
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.bar", &vgw2),
					testAttachmentFunc(&vgw2, &vpc1),
					testAttachmentFunc(&vgw1, &vpc2),
				),
			},
			resource.TestStep{
				Config: testAccCheckVpnGatewayConfigReattach,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.foo", &vgw1),
					testAccCheckVpnGatewayExists(
						"aws_vpn_gateway.bar", &vgw2),
					testAttachmentFunc(&vgw1, &vpc1),
					testAttachmentFunc(&vgw2, &vpc2),
				),
			},
		},
	})
}

func TestAccAWSVpnGateway_delete(t *testing.T) {
	var vpnGateway ec2.VpnGateway

	testDeleted := func(r string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			_, ok := s.RootModule().Resources[r]
			if ok {
				return fmt.Errorf("VPN Gateway %q should have been deleted.", r)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpnGatewayDestroy,
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
	var v ec2.VpnGateway

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_vpn_gateway.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVpnGatewayDestroy,
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

func testAccAWSVpnGatewayDisappears(gateway *ec2.VpnGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		_, err := conn.DetachVpnGateway(&ec2.DetachVpnGatewayInput{
			VpnGatewayId: gateway.VpnGatewayId,
			VpcId:        gateway.VpcAttachments[0].VpcId,
		})
		if err != nil {
			ec2err, ok := err.(awserr.Error)
			if ok {
				if ec2err.Code() == "InvalidVpnGatewayID.NotFound" {
					return nil
				} else if ec2err.Code() == "InvalidVpnGatewayAttachment.NotFound" {
					return nil
				}
			}

			if err != nil {
				return err
			}
		}

		opts := &ec2.DeleteVpnGatewayInput{
			VpnGatewayId: gateway.VpnGatewayId,
		}
		if _, err := conn.DeleteVpnGateway(opts); err != nil {
			return err
		}
		return resource.Retry(40*time.Minute, func() *resource.RetryError {
			opts := &ec2.DescribeVpnGatewaysInput{
				VpnGatewayIds: []*string{gateway.VpnGatewayId},
			}
			resp, err := conn.DescribeVpnGateways(opts)
			if err != nil {
				cgw, ok := err.(awserr.Error)
				if ok && cgw.Code() == "InvalidVpnGatewayID.NotFound" {
					return nil
				}
				if ok && cgw.Code() == "IncorrectState" {
					return resource.RetryableError(fmt.Errorf(
						"Waiting for VPN Gateway to be in the correct state: %v", gateway.VpnGatewayId))
				}
				return resource.NonRetryableError(
					fmt.Errorf("Error retrieving VPN Gateway: %s", err))
			}
			if *resp.VpnGateways[0].State == "deleted" {
				return nil
			}
			return resource.RetryableError(fmt.Errorf(
				"Waiting for VPN Gateway: %v", gateway.VpnGatewayId))
		})
	}
}

func testAccCheckVpnGatewayDestroy(s *terraform.State) error {
	ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpn_gateway" {
			continue
		}

		// Try to find the resource
		resp, err := ec2conn.DescribeVpnGateways(&ec2.DescribeVpnGatewaysInput{
			VpnGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			var v *ec2.VpnGateway
			for _, g := range resp.VpnGateways {
				if *g.VpnGatewayId == rs.Primary.ID {
					v = g
				}
			}

			if v == nil {
				// wasn't found
				return nil
			}

			if *v.State != "deleted" {
				return fmt.Errorf("Expected VPN Gateway to be in deleted state, but was not: %s", v)
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

func testAccCheckVpnGatewayExists(n string, ig *ec2.VpnGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		ec2conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := ec2conn.DescribeVpnGateways(&ec2.DescribeVpnGatewaysInput{
			VpnGatewayIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}
		if len(resp.VpnGateways) == 0 {
			return fmt.Errorf("VPN Gateway not found")
		}

		*ig = *resp.VpnGateways[0]

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

const testAccCheckVpnGatewayConfigReattach = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpc" "bar" {
	cidr_block = "10.2.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_vpn_gateway" "bar" {
	vpc_id = "${aws_vpc.bar.id}"
}
`

const testAccCheckVpnGatewayConfigReattachChange = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpc" "bar" {
	cidr_block = "10.2.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.bar.id}"
}

resource "aws_vpn_gateway" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
}
`

const testAccVpnGatewayConfigWithAZ = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpn_gateway" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	availability_zone = "us-west-2a"
}
`
