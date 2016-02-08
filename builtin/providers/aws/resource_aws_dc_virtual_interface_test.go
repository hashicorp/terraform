package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDCVIRTUALINTERFACE_basic(t *testing.T) {
	var virtualIF directconnect.VirtualInterface

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDCVirtualInterfaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDCVirtualInterfaceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDCVirtualInterfaceExists("aws_dc_virtual_interface.virtualinterface", &virtualIF),
				),
			},
		},
	})
}

func testAccCheckDCVirtualInterfaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).dcconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dc_virtual_interface" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeVirtualInterfaces(&directconnect.DescribeVirtualInterfacesInput{
			VirtualInterfaceId: aws.String(rs.Primary.ID),
		})
		if err == nil {
			if len(resp.VirtualInterfaces) > 0 && strings.ToLower(*resp.VirtualInterfaces[0].VirtualInterfaceState) != "deleted" {
				return fmt.Errorf("still exists")
			}

			return nil
		}

		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckDCVirtualInterfaceExists(n string, ng *directconnect.VirtualInterface) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).dcconn

		resp, err := conn.DescribeVirtualInterfaces(&directconnect.DescribeVirtualInterfacesInput{
			VirtualInterfaceId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}
		if len(resp.VirtualInterfaces) == 0 {
			return fmt.Errorf("DCVirtualInterface not found")
		}

		*ng = *resp.VirtualInterfaces[0]

		return nil
	}
}

const testAccDCVirtualInterfaceConfig = `
resource "aws_vpc" "vpc" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_vpn_gateway" "vpn_gw" {
    vpc_id = "${aws_vpc.vpc.id}"
}


resource "aws_dc_virtual_interface" "vif" {
  connection_id = "dxcon-xyz"
  virtual_interface_name = "test-vif"
  asn = "12345"
  vlan = "1024"
  auth_key = "super_secret_key"
  amazon_address = "10.0.0.45/30"
  customer_address = "10.0.0.46/30"
  interface_type = "private"
  virtual_gateway_id = "${aws_vpn_gateway.vpn_gw.id}"
}
`
