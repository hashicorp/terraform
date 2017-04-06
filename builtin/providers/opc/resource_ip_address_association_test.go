package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCIPAddressAssociation_Basic(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "opc_compute_ip_address_association.test"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIPAddressAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIPAddressAssociationBasic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPAddressAssociationExists,
					resource.TestCheckResourceAttr(
						resourceName, "tags.#", "2"),
				),
			},
			{
				Config: testAccIPAddressAssociationBasic_Update(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName, "tags.#", "1"),
				),
			},
		},
	})
}

func TestAccOPCIPAddressAssociation_Full(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "opc_compute_ip_address_association.test"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIPAddressAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIPAddressAssociationFull(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPAddressAssociationExists,
					resource.TestCheckResourceAttr(
						resourceName, "vnic", fmt.Sprintf("test-vnic-data-%d", rInt)),
					resource.TestCheckResourceAttr(
						resourceName, "ip_address_reservation", fmt.Sprintf("testing-ip-address-association-%d", rInt)),
				),
			},
		},
	})
}

func testAccCheckIPAddressAssociationExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAddressAssociations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_address_association" {
			continue
		}

		input := compute.GetIPAddressAssociationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetIPAddressAssociation(&input); err != nil {
			return fmt.Errorf("Error retrieving state of IP Address Association %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccCheckIPAddressAssociationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAddressAssociations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_address_association" {
			continue
		}

		input := compute.GetIPAddressAssociationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetIPAddressAssociation(&input); err == nil {
			return fmt.Errorf("IP Address Association %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

func testAccIPAddressAssociationBasic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_address_association" "test" {
  name = "testing-acc-%d"
  description = "acctesting ip address association test %d"
  tags = ["tag1", "tag2"]
}`, rInt, rInt)
}

func testAccIPAddressAssociationBasic_Update(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_address_association" "test" {
  name = "testing-acc-%d"
  description = "acctesting ip address association test updated %d"
  tags = ["tag1"]
}`, rInt, rInt)
}

func testAccIPAddressAssociationFull(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "foo" {
  name = "testing-vnic-data-%d"
  description = "testing-ip-address-association"
  ip_address_prefix = "10.1.13.0/24"
}
resource "opc_compute_instance" "test" {
  name = "test-%d"
  label = "test"
  shape = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
  networking_info {
    index = 0
    ip_network = "${opc_compute_ip_network.foo.id}"
    vnic = "test-vnic-data-%d"
    shared_network = false
    mac_address = "02:5a:cd:ec:2e:4c"
  }
}
data "opc_compute_network_interface" "eth0" {
  instance_name = "${opc_compute_instance.test.name}"
  instance_id = "${opc_compute_instance.test.id}"
  interface = "eth0"
}
resource "opc_compute_ip_address_reservation" "test" {
  name = "testing-ip-address-association-%d"
  description = "testing-desc-%d"
  ip_address_pool = "public-ippool"
}
resource "opc_compute_ip_address_association" "test" {
  name = "testing-acc-%d"
  ip_address_reservation = "${opc_compute_ip_address_reservation.test.name}"
  vnic = "${data.opc_compute_network_interface.eth0.vnic}"
  description = "acctesting ip address association test %d"
  tags = ["tag1", "tag2"]
}`, rInt, rInt, rInt, rInt, rInt, rInt, rInt)
}
