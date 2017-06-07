package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsSecurityGroupAttachment_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsSecurityGroupAttachmentConfig(true),
				Check:  checkSecurityGroupAttachment(true),
			},
			resource.TestStep{
				Config: testAccAwsSecurityGroupAttachmentConfig(false),
				Check:  checkSecurityGroupAttachment(false),
			},
		},
	})
}

func testAccAwsSecurityGroupAttachmentConfig(attach bool) string {
	baseConfig := `
data "aws_ami" "ami" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amzn-ami-hvm-*"]
  }

  owners = ["amazon"]
}

resource "aws_instance" "instance" {
  instance_type = "t2.micro"
  ami           = "${data.aws_ami.ami.id}"

  tags = {
    "type" = "terraform-test-instance"
  }
}

resource "aws_security_group" "sg" {
  tags = {
    "type" = "terraform-test-security-group"
  }
}
`
	optionalConfig := `
resource "aws_security_group_attachment" "sg_attachment" {
  security_group_id    = "${aws_security_group.sg.id}"
  network_interface_id = "${aws_instance.instance.primary_network_interface_id}"
}
`

	if attach {
		return baseConfig + optionalConfig
	}
	return baseConfig
}

func checkSecurityGroupAttachment(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		interfaceID := s.Modules[0].Resources["aws_instance.instance"].Primary.Attributes["primary_network_interface_id"]
		sgID := s.Modules[0].Resources["aws_security_group.sg"].Primary.ID

		iface, err := fetchNetworkInterface(conn, interfaceID)
		if err != nil {
			return err
		}
		actual := sgExistsInENI(sgID, iface)
		if expected != actual {
			return fmt.Errorf("expected existence of security group in ENI to be %t, got %t", expected, actual)
		}
		return nil
	}
}
