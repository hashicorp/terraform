package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// TODO (@jake): Add actual vnics after instance resource is finalized
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
				Config: testAccVnicSetBasic(rName, rDesc),
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
				Config: testAccVnicSetBasic_Update(rName, rDesc),
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

// TODO (@jake): add actual vnics once instance resource is finalized
func testAccVnicSetBasic(rName, rDesc string) string {
	return fmt.Sprintf(`
resource "opc_compute_vnic_set" "test" {
  name = "%s"
  description = "%s"
  tags = ["tag1", "tag2"]
  virtual_nics = ["jake-manual_eth1", "jake_manual_two_eth1"]
}`, rName, rDesc)
}

func testAccVnicSetBasic_Update(rName, rDesc string) string {
	return fmt.Sprintf(`
resource "opc_compute_vnic_set" "test" {
  name = "%s"
  description = "%s-updated"
  virtual_nics = ["jake-manual_eth1", "jake_manual_two_eth1"]
  tags = ["tag1"]
}`, rName, rDesc)
}
