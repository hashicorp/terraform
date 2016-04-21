package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDHCPOptionsAssociation_basic(t *testing.T) {
	var v ec2.Vpc
	var d ec2.DhcpOptions

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDHCPOptionsAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDHCPOptionsAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDHCPOptionsExists("aws_vpc_dhcp_options.foo", &d),
					testAccCheckVpcExists("aws_vpc.foo", &v),
					testAccCheckDHCPOptionsAssociationExist("aws_vpc_dhcp_options_association.foo", &v),
				),
			},
		},
	})
}

func testAccCheckDHCPOptionsAssociationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_dhcp_options_association" {
			continue
		}

		// Try to find the VPC associated to the DHCP Options set
		vpcs, err := findVPCsByDHCPOptionsID(conn, rs.Primary.Attributes["dhcp_options_id"])
		if err != nil {
			return err
		}

		if len(vpcs) > 0 {
			return fmt.Errorf("DHCP Options association is still associated to %d VPCs.", len(vpcs))
		}
	}

	return nil
}

func testAccCheckDHCPOptionsAssociationExist(n string, vpc *ec2.Vpc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DHCP Options Set association ID is set")
		}

		if *vpc.DhcpOptionsId != rs.Primary.Attributes["dhcp_options_id"] {
			return fmt.Errorf("VPC %s does not have DHCP Options Set %s associated", *vpc.VpcId, rs.Primary.Attributes["dhcp_options_id"])
		}

		if *vpc.VpcId != rs.Primary.Attributes["vpc_id"] {
			return fmt.Errorf("DHCP Options Set %s is not associated with VPC %s", rs.Primary.Attributes["dhcp_options_id"], *vpc.VpcId)
		}

		return nil
	}
}

const testAccDHCPOptionsAssociationConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_vpc_dhcp_options" "foo" {
	domain_name = "service.consul"
	domain_name_servers = ["127.0.0.1", "10.0.0.2"]
	ntp_servers = ["127.0.0.1"]
	netbios_name_servers = ["127.0.0.1"]
	netbios_node_type = 2

	tags {
		Name = "foo"
	}
}

resource "aws_vpc_dhcp_options_association" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	dhcp_options_id = "${aws_vpc_dhcp_options.foo.id}"
}
`
