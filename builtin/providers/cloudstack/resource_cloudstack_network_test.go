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
					testAccCheckNetworkTags(&network, "terraform-tag", "true"),
				),
			},
		},
	})
}

func TestAccCloudStackNetwork_vpc(t *testing.T) {
	var network cloudstack.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackNetwork_vpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackNetworkExists(
						"cloudstack_network.foo", &network),
					testAccCheckCloudStackNetworkVPCAttributes(&network),
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

		if network.Cidr != CLOUDSTACK_NETWORK_2_CIDR {
			return fmt.Errorf("Bad CIDR: %s", network.Cidr)
		}

		if network.Networkofferingname != CLOUDSTACK_NETWORK_2_OFFERING {
			return fmt.Errorf("Bad network offering: %s", network.Networkofferingname)
		}

		return nil
	}
}

func testAccCheckNetworkTags(
	n *cloudstack.Network, key string, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tags := make(map[string]string)
		for item := range n.Tags {
			tags[n.Tags[item].Key] = n.Tags[item].Value
		}
		return testAccCheckTags(tags, key, value)
	}
}

func testAccCheckCloudStackNetworkVPCAttributes(
	network *cloudstack.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if network.Name != "terraform-network" {
			return fmt.Errorf("Bad name: %s", network.Name)
		}

		if network.Displaytext != "terraform-network" {
			return fmt.Errorf("Bad display name: %s", network.Displaytext)
		}

		if network.Cidr != CLOUDSTACK_VPC_NETWORK_CIDR {
			return fmt.Errorf("Bad CIDR: %s", network.Cidr)
		}

		if network.Networkofferingname != CLOUDSTACK_VPC_NETWORK_OFFERING {
			return fmt.Errorf("Bad network offering: %s", network.Networkofferingname)
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

		_, _, err := cs.Network.GetNetworkByID(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Network %s still exists", rs.Primary.ID)
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
	tags = {
		terraform-tag = "true"
	}
}`,
	CLOUDSTACK_NETWORK_2_CIDR,
	CLOUDSTACK_NETWORK_2_OFFERING,
	CLOUDSTACK_ZONE)

var testAccCloudStackNetwork_vpc = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
	name = "terraform-vpc"
	cidr = "%s"
	vpc_offering = "%s"
	zone = "%s"
}

resource "cloudstack_network" "foo" {
	name = "terraform-network"
	cidr = "%s"
	network_offering = "%s"
	vpc_id = "${cloudstack_vpc.foobar.id}"
	zone = "${cloudstack_vpc.foobar.zone}"
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_VPC_NETWORK_CIDR,
	CLOUDSTACK_VPC_NETWORK_OFFERING)
