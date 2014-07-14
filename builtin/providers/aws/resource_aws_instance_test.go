package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func TestAccAWSInstance(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
				),
			},
		},
	})
}

func TestAccAWSInstance_vpc(t *testing.T) {
	var v ec2.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfigVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceExists(
						"aws_instance.foo", &v),
				),
			},
		},
	})
}

func testAccCheckInstanceDestroy(s *terraform.State) error {
	conn := testAccProvider.ec2conn

	for _, rs := range s.Resources {
		if rs.Type != "aws_instance" {
			continue
		}

		// Try to find the resource
		resp, err := conn.Instances(
			[]string{rs.ID}, ec2.NewFilter())
		if err == nil {
			if len(resp.Reservations) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		if ec2err.Code != "InvalidInstanceID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckInstanceExists(n string, i *ec2.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.ec2conn
		resp, err := conn.Instances(
			[]string{rs.ID}, ec2.NewFilter())
		if err != nil {
			return err
		}
		if len(resp.Reservations) == 0 {
			return fmt.Errorf("Instance not found")
		}

		*i = resp.Reservations[0].Instances[0]

		return nil
	}
}

const testAccInstanceConfig = `
resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
}
`

const testAccInstanceConfigVPC = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "foo" {
	# us-west-2
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	subnet_id = "${aws_subnet.foo.id}"
}
`
