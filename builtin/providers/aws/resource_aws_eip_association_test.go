package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEIPAssociation_basic(t *testing.T) {
	var a ec2.Address

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEIPAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEIPAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEIPExists(
						"aws_eip.bar.0", &a),
					testAccCheckAWSEIPAssociationExists(
						"aws_eip_association.by_allocation_id", &a),
					testAccCheckAWSEIPExists(
						"aws_eip.bar.1", &a),
					testAccCheckAWSEIPAssociationExists(
						"aws_eip_association.by_public_ip", &a),
					testAccCheckAWSEIPExists(
						"aws_eip.bar.2", &a),
					testAccCheckAWSEIPAssociationExists(
						"aws_eip_association.to_eni", &a),
				),
			},
		},
	})
}

func testAccCheckAWSEIPAssociationExists(name string, res *ec2.Address) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EIP Association ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		request := &ec2.DescribeAddressesInput{
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("association-id"),
					Values: []*string{res.AssociationId},
				},
			},
		}
		describe, err := conn.DescribeAddresses(request)
		if err != nil {
			return err
		}

		if len(describe.Addresses) != 1 ||
			*describe.Addresses[0].AssociationId != *res.AssociationId {
			return fmt.Errorf("EIP Association not found")
		}

		return nil
	}
}

func testAccCheckAWSEIPAssociationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_eip_association" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EIP Association ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		request := &ec2.DescribeAddressesInput{
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("association-id"),
					Values: []*string{aws.String(rs.Primary.ID)},
				},
			},
		}
		describe, err := conn.DescribeAddresses(request)
		if err != nil {
			return err
		}

		if len(describe.Addresses) > 0 {
			return fmt.Errorf("EIP Association still exists")
		}
	}
	return nil
}

const testAccAWSEIPAssociationConfig = `
resource "aws_vpc" "main" {
	cidr_block = "192.168.0.0/24"
}
resource "aws_subnet" "sub" {
	vpc_id = "${aws_vpc.main.id}"
	cidr_block = "192.168.0.0/25"
	availability_zone = "us-west-2a"
}
resource "aws_internet_gateway" "igw" {
	vpc_id = "${aws_vpc.main.id}"
}
resource "aws_instance" "foo" {
	count = 2
	ami = "ami-21f78e11"
	availability_zone = "us-west-2a"
	instance_type = "t1.micro"
	subnet_id = "${aws_subnet.sub.id}"
}
resource "aws_eip" "bar" {
	count = 3
	vpc = true
}
resource "aws_eip_association" "by_allocation_id" {
	allocation_id = "${aws_eip.bar.0.id}"
	instance_id = "${aws_instance.foo.0.id}"
}
resource "aws_eip_association" "by_public_ip" {
	public_ip = "${aws_eip.bar.1.public_ip}"
	instance_id = "${aws_instance.foo.1.id}"
}
resource "aws_eip_association" "to_eni" {
	allocation_id = "${aws_eip.bar.2.id}"
	network_interface_id = "${aws_network_interface.baz.id}"
}
resource "aws_network_interface" "baz" {
	subnet_id = "${aws_subnet.sub.id}"
	private_ips = ["192.168.0.10"]
	attachment {
		instance = "${aws_instance.foo.0.id}"
		device_index = 1
	}
}
`
