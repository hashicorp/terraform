package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackIPAddress_basic(t *testing.T) {
	var ipaddr cloudstack.PublicIpAddress

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackIPAddress_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackIPAddressExists(
						"cloudstack_ipaddress.foo", &ipaddr),
					testAccCheckCloudStackIPAddressAttributes(&ipaddr),
				),
			},
		},
	})
}

func TestAccCloudStackIPAddress_vpc(t *testing.T) {
	var ipaddr cloudstack.PublicIpAddress

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackIPAddress_vpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackIPAddressExists(
						"cloudstack_ipaddress.foo", &ipaddr),
				),
			},
		},
	})
}

func testAccCheckCloudStackIPAddressExists(
	n string, ipaddr *cloudstack.PublicIpAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		pip, _, err := cs.Address.GetPublicIpAddressByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if pip.Id != rs.Primary.ID {
			return fmt.Errorf("IP address not found")
		}

		*ipaddr = *pip

		return nil
	}
}

func testAccCheckCloudStackIPAddressAttributes(
	ipaddr *cloudstack.PublicIpAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if ipaddr.Associatednetworkid != CLOUDSTACK_NETWORK_1 {
			return fmt.Errorf("Bad network ID: %s", ipaddr.Associatednetworkid)
		}

		return nil
	}
}

func testAccCheckCloudStackIPAddressDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_ipaddress" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		ip, _, err := cs.Address.GetPublicIpAddressByID(rs.Primary.ID)
		if err == nil && ip.Associatednetworkid != "" {
			return fmt.Errorf("Public IP %s still associated", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackIPAddress_basic = fmt.Sprintf(`
resource "cloudstack_ipaddress" "foo" {
  network_id = "%s"
}`, CLOUDSTACK_NETWORK_1)

var testAccCloudStackIPAddress_vpc = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
  name = "terraform-vpc"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_ipaddress" "foo" {
  vpc_id = "${cloudstack_vpc.foobar.id}"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE)
