package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMVirtualMachineScaleSet_importBasic(t *testing.T) {
	resourceName := "azurerm_virtual_machine_scale_set.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_basic, ri, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_importBasic_managedDisk(t *testing.T) {
	resourceName := "azurerm_virtual_machine_scale_set.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSet_basicLinux_managedDisk, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_importLinux(t *testing.T) {
	resourceName := "azurerm_virtual_machine_scale_set.test"

	ri := acctest.RandInt()
	config := testAccAzureRMVirtualMachineScaleSet_linux(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_importLoadBalancer(t *testing.T) {
	resourceName := "azurerm_virtual_machine_scale_set.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetLoadbalancerTemplate, ri, ri, ri, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_importOverProvision(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetOverprovisionTemplate, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetOverprovision("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_importExtension(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetExtensionTemplate, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetExtension("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineScaleSet_importMultipleExtensions(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVirtualMachineScaleSetMultipleExtensionsTemplate, ri, ri, ri, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineScaleSetExists("azurerm_virtual_machine_scale_set.test"),
					testCheckAzureRMVirtualMachineScaleSetExtension("azurerm_virtual_machine_scale_set.test"),
				),
			},
		},
	})
}
