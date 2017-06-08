package vrealize

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/sky-mah96/govrealize"
)

func TestAccVrealizeMachine_Basic(t *testing.T) {

	machine := govrealize.Machine{
		CatalogItem: govrealize.MachineCatalogItem{
			ID: "c94fa0c3-4aed-43ce-b7a6-4163a07e4cd6",
		},
		Organization: govrealize.MachineOrganization{
			TenantRef:    "vsphere.local",
			SubtenantRef: "f04f060d-73be-48a3-b82c-20cb98efd2d2",
		},
	}

	requestDataMap := map[string]interface{}{
		"Key":   "provider-provisioningGroupId",
		"Value": "f04f060d-73be-48a3-b82c-20cb98efd2d2",
	}

	requestDataMapString := fmt.Sprintf("%v", requestDataMap)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVrealizeMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVrealizeMachineConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVrealizeMachineExists("vrealize_machine.foobar", &machine),
					resource.TestCheckResourceAttr(
						"vrealize_machine.foobar", "catalogItemRefId", "c94fa0c3-4aed-43ce-b7a6-4163a07e4cd6"),
					resource.TestCheckResourceAttr(
						"vrealize_machine.foobar", "tenantRef", "vsphere.local"),
					resource.TestCheckResourceAttr(
						"vrealize_machine.foobar", "subTenantRef", "f04f060d-73be-48a3-b82c-20cb98efd2d2"),
					resource.TestCheckResourceAttr(
						"vrealize_machine.foobar", "requestData", requestDataMapString),
				),
			},
		},
	})
}

const testAccCheckVrealizeMachineConfigBasic = `
resource "machine" "test" {
    catalogItemRefId = "c94fa0c3-4aed-43ce-b7a6-4163a07e4cd6"
    tenantRef = "vsphere.local"
    subTenantRef = "f04f060d-73be-48a3-b82c-20cb98efd2d2"
	requestData = {
        key = "provider-provisioningGroupId"
		value = "f04f060d-73be-48a3-b82c-20cb98efd2d2"
	}
}`

func testAccCheckVrealizeMachineExists(rn string, machine *govrealize.Machine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("not found: %s", rn)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no machine ID is set")
		}

		client := testAccProvider.Meta().(*govrealize.Client)

		got, _, err := client.Machine.GetMachine(rs.Primary.ID)
		if err != nil {
			return err
		}
		if got.ID != machine.ID {
			return fmt.Errorf("wrong machine found, want %q got %q", machine.ID, got.ID)
		}
		// get the computed machine details
		*machine = *got
		return nil
	}
}

func testAccCheckVrealizeMachineDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govrealize.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vrealize_machine" {
			continue
		}

		// Try to find the machine
		_, _, err := client.Machine.GetMachine(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Machine still exists")
		}
	}

	return nil
}
