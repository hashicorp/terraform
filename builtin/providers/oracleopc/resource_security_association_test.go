package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccOPCResourceSecurityAssociation_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: opcResourceCheck(
			associationResourceName,
			testAccCheckAssociationDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityAssociationBasic,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(
						associationResourceName,
						testAccCheckAssociationExists),
				),
			},
		},
	})
}

func testAccCheckAssociationExists(state *OPCResourceState) error {
	associationName := getAssociationName(state)

	if _, err := state.SecurityAssociations().GetSecurityAssociation(associationName); err != nil {
		return fmt.Errorf("Error retrieving state of security assocation %s: %s", associationName, err)
	}

	return nil
}

func getAssociationName(rs *OPCResourceState) string {
	return rs.Attributes["name"]
}

func testAccCheckAssociationDestroyed(state *OPCResourceState) error {
	associationName := getAssociationName(state)
	if info, err := state.SecurityAssociations().GetSecurityAssociation(associationName); err == nil {
		return fmt.Errorf("Association %s still exists: %#v", associationName, info)
	}

	return nil
}

const associationName = "test_rule"

var associationResourceName = fmt.Sprintf("opc_compute_security_association.%s", associationName)

var testAccSecurityAssociationBasic = fmt.Sprintf(`
resource "opc_compute_security_list" "sec-list1" {
	name = "sec-list-1"
        policy = "PERMIT"
        outbound_cidr_policy = "DENY"
}

resource "opc_compute_security_association" "%s" {
	vcable = "${opc_compute_instance.test-instance1.vcable}"
	seclist = "${opc_compute_security_list.sec-list1.name}"
}

resource "opc_compute_instance" "test-instance1" {
	name = "test"
	label = "test"
	shape = "oc3"
	imageList = "/oracle/public/oel_6.4_2GB_v1"
}
`, ruleName)
