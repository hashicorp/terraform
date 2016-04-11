package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackVPNCustomerGateway_basic(t *testing.T) {
	var vpnCustomerGateway cloudstack.VpnCustomerGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackVPNCustomerGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackVPNCustomerGateway_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackVPNCustomerGatewayExists(
						"cloudstack_vpn_customer_gateway.foo", &vpnCustomerGateway),
					testAccCheckCloudStackVPNCustomerGatewayAttributes(&vpnCustomerGateway),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.foo", "name", "terraform-foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.bar", "name", "terraform-bar"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.foo", "ike_policy", "aes256-sha1"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.bar", "esp_policy", "aes256-sha1"),
				),
			},
		},
	})
}

func TestAccCloudStackVPNCustomerGateway_update(t *testing.T) {
	var vpnCustomerGateway cloudstack.VpnCustomerGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackVPNCustomerGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackVPNCustomerGateway_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackVPNCustomerGatewayExists(
						"cloudstack_vpn_customer_gateway.foo", &vpnCustomerGateway),
					testAccCheckCloudStackVPNCustomerGatewayAttributes(&vpnCustomerGateway),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.foo", "name", "terraform-foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.bar", "name", "terraform-bar"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.foo", "ike_policy", "aes256-sha1"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.bar", "esp_policy", "aes256-sha1"),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackVPNCustomerGateway_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackVPNCustomerGatewayExists(
						"cloudstack_vpn_customer_gateway.foo", &vpnCustomerGateway),
					testAccCheckCloudStackVPNCustomerGatewayUpdatedAttributes(&vpnCustomerGateway),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.foo", "name", "terraform-foo-bar"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.bar", "name", "terraform-bar-foo"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.foo", "ike_policy", "3des-md5"),
					resource.TestCheckResourceAttr(
						"cloudstack_vpn_customer_gateway.bar", "esp_policy", "3des-md5"),
				),
			},
		},
	})
}

func testAccCheckCloudStackVPNCustomerGatewayExists(
	n string, vpnCustomerGateway *cloudstack.VpnCustomerGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPN CustomerGateway ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		v, _, err := cs.VPN.GetVpnCustomerGatewayByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if v.Id != rs.Primary.ID {
			return fmt.Errorf("VPN CustomerGateway not found")
		}

		*vpnCustomerGateway = *v

		return nil
	}
}

func testAccCheckCloudStackVPNCustomerGatewayAttributes(
	vpnCustomerGateway *cloudstack.VpnCustomerGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if vpnCustomerGateway.Esppolicy != "aes256-sha1" {
			return fmt.Errorf("Bad ESP policy: %s", vpnCustomerGateway.Esppolicy)
		}

		if vpnCustomerGateway.Ikepolicy != "aes256-sha1" {
			return fmt.Errorf("Bad IKE policy: %s", vpnCustomerGateway.Ikepolicy)
		}

		if vpnCustomerGateway.Ipsecpsk != "terraform" {
			return fmt.Errorf("Bad IPSEC pre-shared key: %s", vpnCustomerGateway.Ipsecpsk)
		}

		return nil
	}
}

func testAccCheckCloudStackVPNCustomerGatewayUpdatedAttributes(
	vpnCustomerGateway *cloudstack.VpnCustomerGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if vpnCustomerGateway.Esppolicy != "3des-md5" {
			return fmt.Errorf("Bad ESP policy: %s", vpnCustomerGateway.Esppolicy)
		}

		if vpnCustomerGateway.Ikepolicy != "3des-md5" {
			return fmt.Errorf("Bad IKE policy: %s", vpnCustomerGateway.Ikepolicy)
		}

		if vpnCustomerGateway.Ipsecpsk != "terraform" {
			return fmt.Errorf("Bad IPSEC pre-shared key: %s", vpnCustomerGateway.Ipsecpsk)
		}

		return nil
	}
}

func testAccCheckCloudStackVPNCustomerGatewayDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_vpn_customer_gateway" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPN Customer Gateway ID is set")
		}

		_, _, err := cs.VPN.GetVpnCustomerGatewayByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("VPN Customer Gateway %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackVPNCustomerGateway_basic = fmt.Sprintf(`
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
	vpc_id = "${cloudstack_vpc.foo.id}"
}

resource "cloudstack_vpn_gateway" "bar" {
	vpc_id = "${cloudstack_vpc.bar.id}"
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
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_VPC_CIDR_2,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)

var testAccCloudStackVPNCustomerGateway_update = fmt.Sprintf(`
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
  vpc_id = "${cloudstack_vpc.foo.id}"
}

resource "cloudstack_vpn_gateway" "bar" {
  vpc_id = "${cloudstack_vpc.bar.id}"
}

resource "cloudstack_vpn_customer_gateway" "foo" {
  name = "terraform-foo-bar"
  cidr = "${cloudstack_vpc.foo.cidr}"
  esp_policy = "3des-md5"
  gateway = "${cloudstack_vpn_gateway.foo.public_ip}"
  ike_policy = "3des-md5"
  ipsec_psk = "terraform"
}

resource "cloudstack_vpn_customer_gateway" "bar" {
  name = "terraform-bar-foo"
  cidr = "${cloudstack_vpc.bar.cidr}"
  esp_policy = "3des-md5"
  gateway = "${cloudstack_vpn_gateway.bar.public_ip}"
  ike_policy = "3des-md5"
  ipsec_psk = "terraform"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_VPC_CIDR_2,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)
