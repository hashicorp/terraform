// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccDataSourceAwsVpnGateway_'
package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAwsVpnGateway_unattached(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsVpnGatewayUnattachedConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.aws_vpn_gateway.test_by_id", "id",
						"aws_vpn_gateway.unattached", "id"),
					resource.TestCheckResourceAttrPair(
						"data.aws_vpn_gateway.test_by_tags", "id",
						"aws_vpn_gateway.unattached", "id"),
					resource.TestCheckResourceAttrSet("data.aws_vpn_gateway.test_by_id", "state"),
					resource.TestCheckResourceAttr("data.aws_vpn_gateway.test_by_tags", "tags.%", "3"),
					resource.TestCheckNoResourceAttr("data.aws_vpn_gateway.test_by_id", "attached_vpc_id"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsVpnGateway_attached(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDataSourceAwsVpnGatewayAttachedConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.aws_vpn_gateway.test_by_attached_vpc_id", "id",
						"aws_vpn_gateway.attached", "id"),
					resource.TestCheckResourceAttrPair(
						"data.aws_vpn_gateway.test_by_attached_vpc_id", "attached_vpc_id",
						"aws_vpc.foo", "id"),
					resource.TestMatchResourceAttr("data.aws_vpn_gateway.test_by_attached_vpc_id", "state", regexp.MustCompile("(?i)available")),
				),
			},
		},
	})
}

func testAccDataSourceAwsVpnGatewayUnattachedConfig(rInt int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpn_gateway" "unattached" {
    tags {
		Name = "terraform-testacc-vpn-gateway-data-source-unattached-%d"
      	ABC  = "testacc-%d"
		XYZ  = "testacc-%d"
    }
}

data "aws_vpn_gateway" "test_by_id" {
	id = "${aws_vpn_gateway.unattached.id}"
}

data "aws_vpn_gateway" "test_by_tags" {
	tags = "${aws_vpn_gateway.unattached.tags}"
}
`, rInt, rInt+1, rInt-1)
}

func testAccDataSourceAwsVpnGatewayAttachedConfig(rInt int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"

  	tags {
    	Name = "terraform-testacc-vpn-gateway-data-source-foo-%d"
  	}
}

resource "aws_vpn_gateway" "attached" {
    tags {
		Name = "terraform-testacc-vpn-gateway-data-source-attached-%d"
    }
}

resource "aws_vpn_gateway_attachment" "vpn_attachment" {
  vpc_id = "${aws_vpc.foo.id}"
  vpn_gateway_id = "${aws_vpn_gateway.attached.id}"
}

data "aws_vpn_gateway" "test_by_attached_vpc_id" {
	attached_vpc_id = "${aws_vpn_gateway_attachment.vpn_attachment.vpc_id}"
}
`, rInt, rInt)
}
