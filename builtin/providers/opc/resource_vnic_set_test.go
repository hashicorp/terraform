package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCVNICSet_Basic(t *testing.T) {
	rInt := acctest.RandInt()
	rName := fmt.Sprintf("testing-acc-%d", rInt)
	rDesc := fmt.Sprintf("acctesting vnic set %d", rInt)
	resourceName := "opc_compute_vnic_set.test"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckVNICSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVnicSetBasic(rName, rDesc, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckVNICSetExists,
					resource.TestCheckResourceAttr(
						resourceName, "name", rName),
					resource.TestCheckResourceAttr(
						resourceName, "description", rDesc),
					resource.TestCheckResourceAttr(
						resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(
						resourceName, "virtual_nics.#", "2"),
				),
			},
			{
				Config: testAccVnicSetBasic_Update(rName, rDesc, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckVNICSetExists,
					resource.TestCheckResourceAttr(
						resourceName, "name", rName),
					resource.TestCheckResourceAttr(
						resourceName, "description", fmt.Sprintf("%s-updated", rDesc)),
					resource.TestCheckResourceAttr(
						resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(
						resourceName, "virtual_nics.#", "2"),
				),
			},
		},
	})
}

func testAccOPCCheckVNICSetExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).VirtNICSets()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_vnic_set" {
			continue
		}

		input := compute.GetVirtualNICSetInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetVirtualNICSet(&input); err != nil {
			return fmt.Errorf("Error retrieving state of VNIC Set %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccOPCCheckVNICSetDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).VirtNICSets()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_vnic_set" {
			continue
		}

		input := compute.GetVirtualNICSetInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetVirtualNICSet(&input); err == nil {
			return fmt.Errorf("VNIC Set %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

func testAccVnicSetBasic(rName, rDesc string, rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "foo" {
  name = "testing-vnic-set-%d"
  description = "testing-vnic-set"
  ip_address_prefix = "10.1.14.0/24"
}

resource "opc_compute_ip_network" "bar" {
  name = "testing-vnic-set2-%d"
  description = "testing-vnic-set2"
  ip_address_prefix = "10.1.15.0/24"
}

resource "opc_compute_instance" "foo" {
  name = "test-vnic-set-%d"
  label = "testing"
  shape = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
  networking_info {
    index = 0
    ip_network = "${opc_compute_ip_network.foo.id}"
    vnic = "test-vnic-set-%d"
    shared_network = false
  }
  networking_info {
    index = 1
    ip_network = "${opc_compute_ip_network.bar.id}"
    vnic = "test-vnic-set2-%d"
    shared_network = false
  }
}

data "opc_compute_network_interface" "foo" {
  instance_name = "${opc_compute_instance.foo.name}"
  instance_id = "${opc_compute_instance.foo.id}"
  interface = "eth0"
}

data "opc_compute_network_interface" "bar" {
  instance_name = "${opc_compute_instance.foo.name}"
  instance_id = "${opc_compute_instance.foo.id}"
  interface = "eth1"
}

resource "opc_compute_vnic_set" "test" {
  name = "%s"
  description = "%s"
  tags = ["tag1", "tag2"]
  virtual_nics = [
    "${data.opc_compute_network_interface.foo.vnic}",
    "${data.opc_compute_network_interface.bar.vnic}",
  ]
}`, rInt, rInt, rInt, rInt, rInt, rName, rDesc)
}

func testAccVnicSetBasic_Update(rName, rDesc string, rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "foo" {
  name = "testing-vnic-set-%d"
  description = "testing-vnic-set"
  ip_address_prefix = "10.1.14.0/24"
}

resource "opc_compute_ip_network" "bar" {
  name = "testing-vnic-set2-%d"
  description = "testing-vnic-set2"
  ip_address_prefix = "10.1.15.0/24"
}

resource "opc_compute_instance" "foo" {
  name = "test-vnic-set-%d"
  label = "testing"
  shape = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
  networking_info {
    index = 0
    ip_network = "${opc_compute_ip_network.foo.id}"
    vnic = "test-vnic-set-%d"
    shared_network = false
  }
  networking_info {
    index = 1
    ip_network = "${opc_compute_ip_network.bar.id}"
    vnic = "test-vnic-set2-%d"
    shared_network = false
  }
}

data "opc_compute_network_interface" "foo" {
  instance_name = "${opc_compute_instance.foo.name}"
  instance_id = "${opc_compute_instance.foo.id}"
  interface = "eth0"
}

data "opc_compute_network_interface" "bar" {
  instance_name = "${opc_compute_instance.foo.name}"
  instance_id = "${opc_compute_instance.foo.id}"
  interface = "eth1"
}

resource "opc_compute_vnic_set" "test" {
  name = "%s"
  description = "%s-updated"
  tags = ["tag1"]
  virtual_nics = [
    "${data.opc_compute_network_interface.foo.vnic}",
    "${data.opc_compute_network_interface.bar.vnic}",
  ]
}`, rInt, rInt, rInt, rInt, rInt, rName, rDesc)
}
