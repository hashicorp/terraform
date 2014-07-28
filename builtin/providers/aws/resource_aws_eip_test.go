package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
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
	conn := testAccProvider.ec2conn

	for _, rs := range s.Resources {
		if rs.Type != "aws_eip" {
			continue
		}

		describe, err := conn.Addresses([]string{rs.ID}, []string{}, nil)

		if err == nil {
			if len(describe.Addresses) != 0 &&
				describe.Addresses[0].PublicIp == rs.ID {
				return fmt.Errorf("EIP still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(*ec2.Error)
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
		if conf.PublicIp == "" {
			return fmt.Errorf("empty public_ip")
		}

		if conf.PrivateIpAddress != "" {
			return fmt.Errorf("should not have private_ip for non-vpc")
		}

		return nil
	}
}

func testAccCheckAWSEIPExists(n string, res *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No EIP ID is set")
		}

		conn := testAccProvider.ec2conn

		if strings.Contains(rs.ID, "eipalloc") {
			describe, err := conn.Addresses([]string{}, []string{rs.ID}, nil)
			if err != nil {
				return err
			}

			if len(describe.Addresses) != 1 ||
				describe.Addresses[0].AllocationId != rs.ID {
				return fmt.Errorf("EIP not found")
			}
			*res = describe.Addresses[0]

		} else {
			describe, err := conn.Addresses([]string{rs.ID}, []string{}, nil)
			if err != nil {
				return err
			}

			if len(describe.Addresses) != 1 ||
				describe.Addresses[0].PublicIp != rs.ID {
				return fmt.Errorf("EIP not found")
			}
			*res = describe.Addresses[0]
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
