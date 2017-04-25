package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCDataSourceNetworkInterface_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "data.opc_compute_network_interface.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetworkInterfaceBasic(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resName, "ip_network", fmt.Sprintf("testing-ip-network-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "vnic", fmt.Sprintf("ip-network-test-%d", rInt)),
					resource.TestCheckResourceAttr(resName, "shared_network", "false"),
				),
			},
		},
	})
}

func TestAccOPCDataSourceNetworkInterface_sharedNetwork(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "data.opc_compute_network_interface.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetworkInterfaceShared(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resName, "nat.#", "1"),
					resource.TestCheckResourceAttr(resName, "shared_network", "true"),
					resource.TestCheckResourceAttr(resName, "sec_lists.#", "1"),
					resource.TestCheckResourceAttr(resName, "name_servers.#", "0"),
					resource.TestCheckResourceAttr(resName, "vnic_sets.#", "0"),
				),
			},
		},
	})
}

func testAccDataSourceNetworkInterfaceBasic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "foo" {
  name = "testing-ip-network-%d"
  description = "testing-ip-network-instance"
  ip_address_prefix = "10.1.12.0/24"
}

resource "opc_compute_instance" "test" {
  name = "test-%d"
  label = "test"
  shape = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
  networking_info {
    index = 0
    ip_network = "${opc_compute_ip_network.foo.id}"
    vnic = "ip-network-test-%d"
    shared_network = false
  }
}

data "opc_compute_network_interface" "test" {
  instance_name = "${opc_compute_instance.test.name}"
  instance_id = "${opc_compute_instance.test.id}"
  interface = "eth0"
}`, rInt, rInt, rInt)
}

func testAccDataSourceNetworkInterfaceShared(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_instance" "test" {
  name = "test-%d"
  label = "test"
  shape = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
  tags = ["tag1", "tag2"]
  networking_info {
    index = 0
    nat = ["ippool:/oracle/public/ippool"]
    shared_network = true
  }
}

data "opc_compute_network_interface" "test" {
  instance_name = "${opc_compute_instance.test.name}"
  instance_id = "${opc_compute_instance.test.id}"
  interface = "eth0"
}`, rInt)
}
