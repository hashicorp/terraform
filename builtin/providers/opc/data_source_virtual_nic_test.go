package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCVNIC_Basic(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccVnicBasic(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.opc_compute_vnic.foo", "mac_address", "02:5a:cd:ec:2e:4c"),
					resource.TestCheckResourceAttr(
						"data.opc_compute_vnic.foo", "transit_flag", "false"),
				),
			},
		},
	})
}

func testAccVnicBasic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_network" "foo" {
  name = "testing-vnic-data-%d"
  description = "testing-vnic-data"
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

data "opc_compute_vnic" "foo" {
  name = "${data.opc_compute_network_interface.eth0.vnic}"
}`, rInt, rInt, rInt)
}
