package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsSecurityGroupAttachment(t *testing.T) {
	cases := []struct {
		Name     string
		External bool
	}{
		{
			Name:     "instance primary interface",
			External: false,
		},
		{
			Name:     "externally supplied instance through data source",
			External: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { testAccPreCheck(t) },
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					resource.TestStep{
						Config: testAccAwsSecurityGroupAttachmentConfig(tc.External, true),
						Check:  checkSecurityGroupAttachment(tc.External, true),
					},
					resource.TestStep{
						Config: testAccAwsSecurityGroupAttachmentConfig(tc.External, false),
						Check:  checkSecurityGroupAttachment(tc.External, false),
					},
				},
			})
		})
	}
}

func testAccAwsSecurityGroupAttachmentConfig(external bool, attach bool) string {
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

data "aws_instance" "external_instance" {
	instance_id = "${aws_instance.instance.id}"
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
  network_interface_id = "${%saws_instance.%sinstance.%snetwork_interface_id}"
}
`

	if attach {
		externalResPre := ""
		externalDataPre := ""
		externalAttrPre := "primary_"
		if external {
			externalResPre = "data."
			externalDataPre = "external_"
			externalAttrPre = ""
		}
		return baseConfig + fmt.Sprintf(optionalConfig, externalResPre, externalDataPre, externalAttrPre)
	}
	return baseConfig
}

func checkSecurityGroupAttachment(external bool, expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		ifAttr := "primary_network_interface_id"
		if external {
			ifAttr = "network_interface_id"
		}
		interfaceID := s.Modules[0].Resources["aws_instance.instance"].Primary.Attributes[ifAttr]
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
