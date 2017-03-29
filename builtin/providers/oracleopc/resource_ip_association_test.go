package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccOPCResourceIPAssociation_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: opcResourceCheck(
			ipAssociationResourceName,
			testAccCheckIPAssociationDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccIPAssociationBasic,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(
						ipAssociationResourceName,
						testAccCheckIPAssociationExists),
				),
			},
		},
	})
}

func testAccCheckIPAssociationExists(state *OPCResourceState) error {
	associationName := getIPAssociationName(state)

	if _, err := state.IPAssociations().GetIPAssociation(associationName); err != nil {
		return fmt.Errorf("Error retrieving state of ip assocation %s: %s", associationName, err)
	}

	return nil
}

func getIPAssociationName(rs *OPCResourceState) string {
	return rs.Attributes["name"]
}

func testAccCheckIPAssociationDestroyed(state *OPCResourceState) error {
	associationName := getAssociationName(state)
	if info, err := state.IPAssociations().GetIPAssociation(associationName); err == nil {
		return fmt.Errorf("IP association %s still exists: %#v", associationName, info)
	}

	return nil
}

const ipAssociationName = "test_ip_association"

var ipAssociationResourceName = fmt.Sprintf("opc_compute_ip_association.%s", ipAssociationName)

var testAccIPAssociationBasic = fmt.Sprintf(`
resource "opc_compute_ip_reservation" "reservation1" {
        parentpool = "/oracle/public/ippool"
        permanent = true
}

resource "opc_compute_ip_association" "%s" {
	vcable = "${opc_compute_instance.test-instance1.vcable}"
	parentpool = "ipreservation:${opc_compute_ip_reservation.reservation1.name}"
}

resource "opc_compute_instance" "test-instance1" {
	name = "test"
	label = "test"
	shape = "oc3"
	imageList = "/oracle/public/oel_6.4_2GB_v1"
}
`, ipAssociationName)
