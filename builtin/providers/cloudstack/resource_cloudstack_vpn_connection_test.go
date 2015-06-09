package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackVPNConnection_basic(t *testing.T) {
	var vpnConnection cloudstack.VpnConnection

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackVPNConnectionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackVPNConnection_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackVPNConnectionExists(
						"cloudstack_vpn_connection.foo-bar", &vpnConnection),
					testAccCheckCloudStackVPNConnectionExists(
						"cloudstack_vpn_connection.bar-foo", &vpnConnection),
				),
			},
		},
	})
}

func testAccCheckCloudStackVPNConnectionExists(
	n string, vpnConnection *cloudstack.VpnConnection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPN Connection ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		v, _, err := cs.VPN.GetVpnConnectionByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if v.Id != rs.Primary.ID {
			return fmt.Errorf("VPN Connection not found")
		}

		*vpnConnection = *v

		return nil
	}
}

func testAccCheckCloudStackVPNConnectionDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_vpn_connection" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPN Connection ID is set")
		}

		_, _, err := cs.VPN.GetVpnConnectionByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("VPN Connection %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackVPNConnection_basic = fmt.Sprintf(`
resource "cloudstack_vpc" "foo" {
  name = "terraform-vpc-foo"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_vpc" "bar" {
  name = "terraform-vpc-bar"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_vpn_gateway" "foo" {
  vpc = "${cloudstack_vpc.foo.name}"
}

resource "cloudstack_vpn_gateway" "bar" {
  vpc = "${cloudstack_vpc.bar.name}"
}

resource "cloudstack_vpn_customer_gateway" "foo" {
  name = "terraform-foo"
  cidr = "${cloudstack_vpc.foo.cidr}"
  esp_policy = "aes256-sha1"
  gateway = "${cloudstack_vpn_gateway.foo.public_ip}"
  ike_policy = "aes256-sha1"
  ipsec_psk = "terraform"
}

resource "cloudstack_vpn_customer_gateway" "bar" {
  name = "terraform-bar"
  cidr = "${cloudstack_vpc.bar.cidr}"
  esp_policy = "aes256-sha1"
  gateway = "${cloudstack_vpn_gateway.bar.public_ip}"
  ike_policy = "aes256-sha1"
  ipsec_psk = "terraform"
}

resource "cloudstack_vpn_connection" "foo-bar" {
  customergatewayid = "${cloudstack_vpn_customer_gateway.foo.id}"
  vpngatewayid = "${cloudstack_vpn_gateway.bar.id}"
}

resource "cloudstack_vpn_connection" "bar-foo" {
  customergatewayid = "${cloudstack_vpn_customer_gateway.bar.id}"
  vpngatewayid = "${cloudstack_vpn_gateway.foo.id}"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_VPC_CIDR_2,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)
