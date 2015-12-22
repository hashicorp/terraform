package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDHCPOptions_basic(t *testing.T) {
	var d ec2.DhcpOptions

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDHCPOptionsDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDHCPOptionsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDHCPOptionsExists("aws_vpc_dhcp_options.foo", &d),
					resource.TestCheckResourceAttr("aws_vpc_dhcp_options.foo", "domain_name", "service.consul"),
					resource.TestCheckResourceAttr("aws_vpc_dhcp_options.foo", "domain_name_servers.0", "127.0.0.1"),
					resource.TestCheckResourceAttr("aws_vpc_dhcp_options.foo", "domain_name_servers.1", "10.0.0.2"),
					resource.TestCheckResourceAttr("aws_vpc_dhcp_options.foo", "ntp_servers.0", "127.0.0.1"),
					resource.TestCheckResourceAttr("aws_vpc_dhcp_options.foo", "netbios_name_servers.0", "127.0.0.1"),
					resource.TestCheckResourceAttr("aws_vpc_dhcp_options.foo", "netbios_node_type", "2"),
					resource.TestCheckResourceAttr("aws_vpc_dhcp_options.foo", "tags.Name", "foo-name"),
				),
			},
		},
	})
}

func testAccCheckDHCPOptionsDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_dhcp_options" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeDhcpOptions(&ec2.DescribeDhcpOptionsInput{
			DhcpOptionsIds: []*string{
				aws.String(rs.Primary.ID),
			},
		})
		if ae, ok := err.(awserr.Error); ok && ae.Code() == "InvalidDhcpOptionID.NotFound" {
			continue
		}
		if err == nil {
			if len(resp.DhcpOptions) > 0 {
				return fmt.Errorf("still exists")
			}

			return nil
		}

		// Verify the error is what we want
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if ec2err.Code() != "InvalidDhcpOptionsID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckDHCPOptionsExists(n string, d *ec2.DhcpOptions) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeDhcpOptions(&ec2.DescribeDhcpOptionsInput{
			DhcpOptionsIds: []*string{
				aws.String(rs.Primary.ID),
			},
		})
		if err != nil {
			return err
		}
		if len(resp.DhcpOptions) == 0 {
			return fmt.Errorf("DHCP Options not found")
		}

		*d = *resp.DhcpOptions[0]

		return nil
	}
}

const testAccDHCPOptionsConfig = `
resource "aws_vpc_dhcp_options" "foo" {
	domain_name = "service.consul"
	domain_name_servers = ["127.0.0.1", "10.0.0.2"]
	ntp_servers = ["127.0.0.1"]
	netbios_name_servers = ["127.0.0.1"]
	netbios_node_type = 2

	tags {
		Name = "foo-name"
	}
}
`
