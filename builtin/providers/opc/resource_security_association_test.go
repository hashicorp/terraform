package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCSecurityAssociation_Basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccSecurityAssociationBasic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckSecurityAssociationExists,
				),
			},
		},
	})
}

func TestAccOPCSecurityAssociation_Complete(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccSecurityAssociationComplete, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckSecurityAssociationExists,
				),
			},
		},
	})
}

func testAccOPCCheckSecurityAssociationExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityAssociations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_association" {
			continue
		}

		input := compute.GetSecurityAssociationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetSecurityAssociation(&input); err != nil {
			return fmt.Errorf("Error retrieving state of Security Association %s: %s", input.Name, err)
		}
	}

	return nil
}

func testAccOPCCheckSecurityAssociationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).SecurityAssociations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_security_association" {
			continue
		}

		input := compute.GetSecurityAssociationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetSecurityAssociation(&input); err == nil {
			return fmt.Errorf("Security Association %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

var testAccSecurityAssociationBasic = `
resource "opc_compute_security_list" "test" {
  name                 = "acc-test-sec-ass-sec-list-%d"
  policy               = "PERMIT"
  outbound_cidr_policy = "DENY"
}

resource "opc_compute_instance" "test" {
  name        = "acc-test-sec-ass-instance-%d"
  label       = "Security Associations Test Instance"
  shape       = "oc3"
  image_list   = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
}

resource "opc_compute_security_association" "test" {
  vcable  = "${opc_compute_instance.test.vcable}"
  seclist = "${opc_compute_security_list.test.name}"
}
`

var testAccSecurityAssociationComplete = `
resource "opc_compute_security_list" "test" {
  name                 = "acc-test-sec-ass-sec-list-%d"
  policy               = "PERMIT"
  outbound_cidr_policy = "DENY"
}

resource "opc_compute_instance" "test" {
  name        = "acc-test-sec-ass-instance-%d"
  label       = "Security Associations Test Instance"
  shape       = "oc3"
  image_list   = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"
}

resource "opc_compute_security_association" "test" {
  name    = "acc-test-sec-ass-%d"
  vcable  = "${opc_compute_instance.test.vcable}"
  seclist = "${opc_compute_security_list.test.name}"
}
`
