package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackStaticRoute_basic(t *testing.T) {
	var staticroute cloudstack.StaticRoute

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackStaticRouteDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackStaticRoute_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackStaticRouteExists(
						"cloudstack_static_route.bar", &staticroute),
					testAccCheckCloudStackStaticRouteAttributes(&staticroute),
				),
			},
		},
	})
}

func testAccCheckCloudStackStaticRouteExists(
	n string, staticroute *cloudstack.StaticRoute) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Static Route ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		route, _, err := cs.VPC.GetStaticRouteByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if route.Id != rs.Primary.ID {
			return fmt.Errorf("Static Route not found")
		}

		*staticroute = *route

		return nil
	}
}

func testAccCheckCloudStackStaticRouteAttributes(
	staticroute *cloudstack.StaticRoute) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if staticroute.Cidr != CLOUDSTACK_STATIC_ROUTE_CIDR {
			return fmt.Errorf("Bad Cidr: %s", staticroute.Cidr)
		}

		return nil
	}
}

func testAccCheckCloudStackStaticRouteDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_static_route" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No static route ID is set")
		}

		staticroute, _, err := cs.VPC.GetStaticRouteByID(rs.Primary.ID)
		if err == nil && staticroute.Id != "" {
			return fmt.Errorf("Static route %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

var testAccCloudStackStaticRoute_basic = fmt.Sprintf(`
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
}

resource "cloudstack_static_route" "bar" {
  cidr = "%s"
  gateway_id = "${cloudstack_private_gateway.foo.id}"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_PRIVGW_GATEWAY,
	CLOUDSTACK_PRIVGW_IPADDRESS,
	CLOUDSTACK_PRIVGW_NETMASK,
	CLOUDSTACK_PRIVGW_VLAN,
	CLOUDSTACK_STATIC_ROUTE_CIDR)
