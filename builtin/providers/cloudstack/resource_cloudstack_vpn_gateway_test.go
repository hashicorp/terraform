package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackVPNGateway_basic(t *testing.T) {
	var vpnGateway cloudstack.VpnGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackVPNGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackVPNGateway_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackVPNGatewayExists(
						"cloudstack_vpn_gateway.foo", &vpnGateway),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_gateway.foo", "vpc", "terraform-vpc"),
				),
			},
		},
	})
}

func testAccCheckCloudStackVPNGatewayExists(
	n string, vpnGateway *cloudstack.VpnGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPN Gateway ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		v, _, err := cs.VPN.GetVpnGatewayByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if v.Id != rs.Primary.ID {
			return fmt.Errorf("VPN Gateway not found")
		}

		*vpnGateway = *v

		return nil
	}
}

func testAccCheckCloudStackVPNGatewayDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_vpn_gateway" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPN Gateway ID is set")
		}

		_, _, err := cs.VPN.GetVpnGatewayByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("VPN Gateway %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackVPNGateway_basic = fmt.Sprintf(`
resource "cloudstack_vpc" "foo" {
  name = "terraform-vpc"
  display_text = "terraform-vpc-text"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_vpn_gateway" "foo" {
  vpc = "${cloudstack_vpc.foo.name}"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)
