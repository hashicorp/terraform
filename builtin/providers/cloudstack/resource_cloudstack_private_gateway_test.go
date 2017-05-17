package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackPrivateGateway_basic(t *testing.T) {
	var gateway cloudstack.PrivateGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackPrivateGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackPrivateGateway_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackPrivateGatewayExists(
						"cloudstack_private_gateway.foo", &gateway),
					testAccCheckCloudStackPrivateGatewayAttributes(&gateway),
				),
			},
		},
	})
}

func testAccCheckCloudStackPrivateGatewayExists(
	n string, gateway *cloudstack.PrivateGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Private Gateway ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		pgw, _, err := cs.VPC.GetPrivateGatewayByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if pgw.Id != rs.Primary.ID {
			return fmt.Errorf("Private Gateway not found")
		}

		*gateway = *pgw

		return nil
	}
}

func testAccCheckCloudStackPrivateGatewayAttributes(
	gateway *cloudstack.PrivateGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if gateway.Gateway != CLOUDSTACK_PRIVGW_GATEWAY {
			return fmt.Errorf("Bad Gateway: %s", gateway.Gateway)
		}

		if gateway.Ipaddress != CLOUDSTACK_PRIVGW_IPADDRESS {
			return fmt.Errorf("Bad Gateway: %s", gateway.Ipaddress)
		}

		if gateway.Netmask != CLOUDSTACK_PRIVGW_NETMASK {
			return fmt.Errorf("Bad Gateway: %s", gateway.Netmask)
		}

		return nil
	}
}

func testAccCheckCloudStackPrivateGatewayDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_private_gateway" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No private gateway ID is set")
		}

		gateway, _, err := cs.VPC.GetPrivateGatewayByID(rs.Primary.ID)
		if err == nil && gateway.Id != "" {
			return fmt.Errorf("Private gateway %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackPrivateGateway_basic = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
  name = "terraform-vpc"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_private_gateway" "foo" {
  gateway = "%s"
  ip_address = "%s"
  netmask = "%s"
  vlan = "%s"
  vpc_id = "${cloudstack_vpc.foobar.id}"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_PRIVGW_GATEWAY,
	CLOUDSTACK_PRIVGW_IPADDRESS,
	CLOUDSTACK_PRIVGW_NETMASK,
	CLOUDSTACK_PRIVGW_VLAN)
