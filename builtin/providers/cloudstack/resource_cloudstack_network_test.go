package cloudstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackNetwork_basic(t *testing.T) {
	var network cloudstack.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNetwork_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNetworkExists(
						"cloudstack_network.foo", &network),
					testAccCheckCloudStackNetworkBasicAttributes(&network),
				),
			},
		},
	})
}

func TestAccCloudStackNetwork_vpcACL(t *testing.T) {
	var network cloudstack.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNetwork_vpcACL,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNetworkExists(
						"cloudstack_network.foo", &network),
					testAccCheckCloudStackNetworkVPCACLAttributes(&network),
					resource.TestCheckResourceAttr(
						"cloudstack_network.foo", "vpc", "terraform-vpc"),
				),
			},
		},
	})
}

func testAccCheckCloudStackNetworkExists(
	n string, network *cloudstack.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No network ID is set")
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		ntwrk, _, err := cs.Network.GetNetworkByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if ntwrk.Id != rs.Primary.ID {
			return fmt.Errorf("Network not found")
		}

		*network = *ntwrk

		return nil
	}
}

func testAccCheckCloudStackNetworkBasicAttributes(
	network *cloudstack.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if network.Name != "terraform-network" {
			return fmt.Errorf("Bad name: %s", network.Name)
		}

		if network.Displaytext != "terraform-network" {
			return fmt.Errorf("Bad display name: %s", network.Displaytext)
		}

		if network.Cidr != CLOUDSTACK_NETWORK_1_CIDR {
			return fmt.Errorf("Bad service offering: %s", network.Cidr)
		}

		if network.Networkofferingname != CLOUDSTACK_NETWORK_1_OFFERING {
			return fmt.Errorf("Bad template: %s", network.Networkofferingname)
		}

		return nil
	}
}

func testAccCheckCloudStackNetworkVPCACLAttributes(
	network *cloudstack.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if network.Name != "terraform-network" {
			return fmt.Errorf("Bad name: %s", network.Name)
		}

		if network.Displaytext != "terraform-network" {
			return fmt.Errorf("Bad display name: %s", network.Displaytext)
		}

		if network.Cidr != CLOUDSTACK_VPC_NETWORK_CIDR {
			return fmt.Errorf("Bad service offering: %s", network.Cidr)
		}

		if network.Networkofferingname != CLOUDSTACK_VPC_NETWORK_OFFERING {
			return fmt.Errorf("Bad template: %s", network.Networkofferingname)
		}

		return nil
	}
}

func testAccCheckCloudStackNetworkDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_network" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No network ID is set")
		}

		p := cs.Network.NewDeleteNetworkParams(rs.Primary.ID)
		err, _ := cs.Network.DeleteNetwork(p)

		if err != nil {
			return fmt.Errorf(
				"Error deleting network (%s): %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

var testAccCloudStackNetwork_basic = fmt.Sprintf(`
resource "cloudstack_network" "foo" {
  name = "terraform-network"
  cidr = "%s"
  network_offering = "%s"
  zone = "%s"
}`,
	CLOUDSTACK_NETWORK_1_CIDR,
	CLOUDSTACK_NETWORK_1_OFFERING,
	CLOUDSTACK_ZONE)

var testAccCloudStackNetwork_vpcACL = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
	name = "terraform-vpc"
	cidr = "%s"
	vpc_offering = "%s"
	zone = "%s"
}

resource "cloudstack_network_acl" "foo" {
  name = "terraform-acl"
  description = "terraform-acl-text"
  vpc = "${cloudstack_vpc.foobar.name}"
}

resource "cloudstack_network" "foo" {
  name = "terraform-network"
  cidr = "%s"
  network_offering = "%s"
  vpc = "${cloudstack_vpc.foobar.name}"
  aclid = "${cloudstack_network_acl.foo.id}"
  zone = "${cloudstack_vpc.foobar.zone}"
}`,
	CLOUDSTACK_VPC_CIDR,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_VPC_NETWORK_CIDR,
	CLOUDSTACK_VPC_NETWORK_OFFERING)
