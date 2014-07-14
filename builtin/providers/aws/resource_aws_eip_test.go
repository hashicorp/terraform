package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccAWSEIP(t *testing.T) {
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

func testAccCheckAWSEIPDestroy(s *terraform.State) error {
	conn := testAccProvider.ec2conn

	for _, rs := range s.Resources {
		if rs.Type != "aws_eip" {
			continue

		describe, err := ec2conn.Addresses([]string{}, []string{rs.ID}, nil)

		if err == nil {
			if len(describeGroups.EIPs) != 0 &&
				describeGroups.EIPs[0].Name == rs.ID {
				return fmt.Errorf("EIP still exists")
			}
		}

		// Verify the error
		providerErr, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		if providerErr.Code != "InvalidEIP.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSEIPAttributes(conf *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if conf.PublicIp == "" {
			return fmt.Errorf("empty public_ip")
		}

		if conf.PrivateIpAddress == "" {
			return fmt.Errorf("empty private_ip")
		}

		if conf.InstanceId == "" {
			return fmt.Errorf("empty instance_id")
		}

		return nil
	}
}

func testAccCheckAWSEIPExists(n string, res *ec2.EIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No EIP ID is set")
		}

		conn := testAccProvider.ec2conn

		describeOpts := ec2.DescribeEIPs{
			Names: []string{rs.ID},
		}
		describe, err := conn.DescribeEIPs(&describeOpts)

		if err != nil {
			return err
		}

		if len(describe.EIPs) != 1 ||
			describe.EIPs[0].Name != rs.ID {
			return fmt.Errorf("EIP Group not found")
		}

		*res = describe.EIPs[0]

		return nil
	}
}

const testAccAWSEIPConfig = `
resource "aws_eip" "bar" {
  name = "foobar-terraform-test"
  image_id = "ami-fb8e9292"
  instance_type = "t1.micro"
}
`
