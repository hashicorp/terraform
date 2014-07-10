package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccVpc(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.foo"),
					testAccCheckVpcCidr("aws_vpc.foo", "10.1.0.0/16"),
				),
			},
		},
	})
}

func testAccCheckVpcDestroy(s *terraform.State) error {
	conn := testAccProvider.ec2conn

	for _, rs := range s.Resources {
		if rs.Type != "aws_vpc" {
			continue
		}

		// Try to find the VPC
		resp, err := conn.DescribeVpcs([]string{rs.ID}, ec2.NewFilter())
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

func testAccCheckVpcCidr(n, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		v, ok := rs.Attributes["cidr_block"]
		if !ok {
			return fmt.Errorf("No cidr_block")
		}

		if v != expected {
			return fmt.Errorf("Bad cidr: %s", v)
		}

		return nil
	}
}

func testAccCheckVpcExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No VPC ID is set")
		}

		conn := testAccProvider.ec2conn
		resp, err := conn.DescribeVpcs([]string{rs.ID}, ec2.NewFilter())
		if err != nil {
			return err
		}
		if len(resp.VPCs) == 0 {
			return fmt.Errorf("VPC not found")
		}

		return nil
	}
}

const testAccVpcConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}
`
