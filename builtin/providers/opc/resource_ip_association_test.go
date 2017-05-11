package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCIPAssociation_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccIPAssociationBasic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckIPAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckIPAssociationExists,
				),
			},
		},
	})
}

func testAccOPCCheckIPAssociationExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAssociations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_association" {
			continue
		}

		input := compute.GetIPAssociationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetIPAssociation(&input); err != nil {
			return fmt.Errorf("Error retrieving state of IP Association %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccOPCCheckIPAssociationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAssociations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_association" {
			continue
		}

		input := compute.GetIPAssociationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetIPAssociation(&input); err == nil {
			return fmt.Errorf("IP Association %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

var testAccIPAssociationBasic = `
resource "opc_compute_instance" "test" {
  name      = "test-acc-ip-ass-instance-%d"
  label     = "testAccIPAssociationBasic"
  shape     = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
}

resource "opc_compute_ip_reservation" "test" {
  name        = "test-acc-ip-ass-reservation-%d"
  parent_pool = "/oracle/public/ippool"
  permanent   = true
}

resource "opc_compute_ip_association" "test" {
  vcable      = "${opc_compute_instance.test.vcable}"
  parent_pool = "ipreservation:${opc_compute_ip_reservation.test.name}"
}
`
