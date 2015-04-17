package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEIP_normal(t *testing.T) {
	var conf ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &conf),
					testAccCheckAWSEIPAttributes(&conf),
				),
			},
		},
	})
}

func TestAccAWSEIP_instance(t *testing.T) {
	var conf ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &conf),
					testAccCheckAWSEIPAttributes(&conf),
				),
			},

			resource.TestStep{
				Config: testAccAWSEIPInstanceConfig2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists("aws_eip.bar", &conf),
					testAccCheckAWSEIPAttributes(&conf),
				),
			},
		},
	})
}

func testAccCheckAWSEIPDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_eip" {
			continue
		}

		req := &ec2.DescribeAddressesInput{
			AllocationIDs: []*string{},
			PublicIPs:     []*string{aws.String(rs.Primary.ID)},
		}
		describe, err := conn.DescribeAddresses(req)

		if err == nil {
			if len(describe.Addresses) != 0 &&
				*describe.Addresses[0].PublicIP == rs.Primary.ID {
				return fmt.Errorf("EIP still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(aws.APIError)
		if !ok {
			return err
		}

		if providerErr.Code != "InvalidAllocationID.NotFound" {
			return fmt.Errorf("Unexpected error: %s", err)
		}
	}

	return nil
}

func testAccCheckAWSEIPAttributes(conf *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *conf.PublicIP == "" {
			return fmt.Errorf("empty public_ip")
		}

		return nil
	}
}

func testAccCheckAWSEIPExists(n string, res *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EIP ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		if strings.Contains(rs.Primary.ID, "eipalloc") {
			req := &ec2.DescribeAddressesInput{
				AllocationIDs: []*string{aws.String(rs.Primary.ID)},
				PublicIPs:     []*string{},
			}
			describe, err := conn.DescribeAddresses(req)
			if err != nil {
				return err
			}

			if len(describe.Addresses) != 1 ||
				*describe.Addresses[0].AllocationID != rs.Primary.ID {
				return fmt.Errorf("EIP not found")
			}
			*res = *describe.Addresses[0]

		} else {
			req := &ec2.DescribeAddressesInput{
				AllocationIDs: []*string{},
				PublicIPs:     []*string{aws.String(rs.Primary.ID)},
			}
			describe, err := conn.DescribeAddresses(req)
			if err != nil {
				return err
			}

			if len(describe.Addresses) != 1 ||
				*describe.Addresses[0].PublicIP != rs.Primary.ID {
				return fmt.Errorf("EIP not found")
			}
			*res = *describe.Addresses[0]
		}

		return nil
	}
}

const testAccAWSEIPConfig = `
resource "aws_eip" "bar" {
}
`

const testAccAWSEIPInstanceConfig = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
}

resource "aws_eip" "bar" {
	instance = "${aws_instance.foo.id}"
}
`

const testAccAWSEIPInstanceConfig2 = `
resource "aws_instance" "bar" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
}

resource "aws_eip" "bar" {
	instance = "${aws_instance.bar.id}"
}
`
