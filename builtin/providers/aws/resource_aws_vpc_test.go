package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccVpc(t *testing.T) {
	var vpc ec2.VPC

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					testAccCheckVpcCidr(&vpc, "10.1.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_vpc.foo", "cidr_block", "10.1.0.0/16"),
				),
			},
		},
	})
}

func TestAccVpcUpdate(t *testing.T) {
	var vpc ec2.VPC

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					testAccCheckVpcCidr(&vpc, "10.1.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_vpc.foo", "cidr_block", "10.1.0.0/16"),
				),
			},
			resource.TestStep{
				Config: testAccVpcConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo", &vpc),
					resource.TestCheckResourceAttr(
						"aws_vpc.foo", "enable_dns_hostnames", "true"),
				),
			},
		},
	})
}

func testAccCheckVpcDestroy(s *terraform.State) error {
	conn := testAccProvider.ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc" {
			continue
		}

		// Try to find the VPC
		resp, err := conn.DescribeVpcs([]string{rs.Primary.ID}, ec2.NewFilter())
		if err == nil {
			if len(resp.VPCs) > 0 {
				return fmt.Errorf("VPCs still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		if ec2err.Code != "InvalidVpcID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckVpcCidr(vpc *ec2.VPC, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if vpc.CidrBlock != expected {
			return fmt.Errorf("Bad cidr: %s", vpc.CidrBlock)
		}

		return nil
	}
}

func testAccCheckVpcExists(n string, vpc *ec2.VPC) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPC ID is set")
		}

		conn := testAccProvider.ec2conn
		resp, err := conn.DescribeVpcs([]string{rs.Primary.ID}, ec2.NewFilter())
		if err != nil {
			return err
		}
		if len(resp.VPCs) == 0 {
			return fmt.Errorf("VPC not found")
		}

		*vpc = resp.VPCs[0]

		return nil
	}
}

const testAccVpcConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
`

const testAccVpcConfigUpdate = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	enable_dns_hostnames = true
}
`
