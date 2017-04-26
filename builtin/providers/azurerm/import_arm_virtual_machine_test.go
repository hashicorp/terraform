package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMVirtualMachine_importBasic(t *testing.T) {
	resourceName := "azurerm_virtual_machine.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"delete_data_disks_on_termination",
					"delete_os_disk_on_termination",
				},
			},
		},
	})
}

func TestAccAzureRMVirtualMachine_importBasic_managedDisk(t *testing.T) {
	resourceName := "azurerm_virtual_machine.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine_managedDisk_explicit, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"delete_data_disks_on_termination",
					"delete_os_disk_on_termination",
				},
			},
		},
	})
}
