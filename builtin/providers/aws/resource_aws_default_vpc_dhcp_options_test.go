// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccAWSDefaultVpc_'
package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDefaultVpcDhcpOptions_basic(t *testing.T) {
	var d ec2.DhcpOptions

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDefaultVpcDhcpOptionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDefaultVpcDhcpOptionsConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDHCPOptionsExists("aws_default_vpc_dhcp_options.foo", &d),
					resource.TestCheckResourceAttr(
						"aws_default_vpc_dhcp_options.foo", "domain_name", "us-west-2.compute.internal"),
					resource.TestCheckResourceAttr(
						"aws_default_vpc_dhcp_options.foo", "domain_name_servers", "AmazonProvidedDNS"),
					resource.TestCheckResourceAttr(
						"aws_default_vpc_dhcp_options.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"aws_default_vpc_dhcp_options.foo", "tags.Name", "Default DHCP Option Set"),
				),
			},
		},
	})
}

func testAccCheckAWSDefaultVpcDhcpOptionsDestroy(s *terraform.State) error {
	// We expect DHCP Options Set to still exist
	return nil
}

const testAccAWSDefaultVpcDhcpOptionsConfigBasic = `
provider "aws" {
    region = "us-west-2"
}

resource "aws_default_vpc_dhcp_options" "foo" {
	tags {
		Name = "Default DHCP Option Set"
	}
}
`
