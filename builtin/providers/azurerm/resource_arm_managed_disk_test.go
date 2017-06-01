package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/disk"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMManagedDisk_empty(t *testing.T) {
	var d disk.Model
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMManagedDisk_empty, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedDiskDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedDiskExists("azurerm_managed_disk.test", &d, true),
				),
			},
		},
	})
}

func TestAccAzureRMManagedDisk_import(t *testing.T) {
	var d disk.Model
	var vm compute.VirtualMachine
	ri := acctest.RandInt()
	vmConfig := fmt.Sprintf(testAccAzureRMVirtualMachine_basicLinuxMachine, ri, ri, ri, ri, ri, ri, ri)
	config := fmt.Sprintf(testAccAzureRMManagedDisk_import, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedDiskDestroy,
		Steps: []resource.TestStep{
			{
				//need to create a vm and then delete it so we can use the vhd to test import
				Config:             vmConfig,
				Destroy:            false,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExists("azurerm_virtual_machine.test", &vm),
					testDeleteAzureRMVirtualMachine("azurerm_virtual_machine.test"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedDiskExists("azurerm_managed_disk.test", &d, true),
				),
			},
		},
	})
}

func TestAccAzureRMManagedDisk_copy(t *testing.T) {
	var d disk.Model
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMManagedDisk_copy, ri, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedDiskDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedDiskExists("azurerm_managed_disk.test", &d, true),
				),
			},
		},
	})
}

func TestAccAzureRMManagedDisk_update(t *testing.T) {
	var d disk.Model

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMManagedDisk_empty, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMManagedDisk_empty_updated, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedDiskDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedDiskExists("azurerm_managed_disk.test", &d, true),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "tags.environment", "acctest"),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "tags.cost-center", "ops"),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "disk_size_gb", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "storage_account_type", string(disk.StandardLRS)),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedDiskExists("azurerm_managed_disk.test", &d, true),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "tags.environment", "acctest"),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "disk_size_gb", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_managed_disk.test", "storage_account_type", string(disk.PremiumLRS)),
				),
			},
		},
	})
}

func TestAccAzureRMManagedDisk_NonStandardCasing(t *testing.T) {
	var d disk.Model
	ri := acctest.RandInt()
	config := testAccAzureRMManagedDiskNonStandardCasing(ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMManagedDiskExists("azurerm_managed_disk.test", &d, true),
				),
			},
			resource.TestStep{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testCheckAzureRMManagedDiskExists(name string, d *disk.Model, shouldExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		dName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for disk: %s", dName)
		}

		conn := testAccProvider.Meta().(*ArmClient).diskClient

		resp, err := conn.Get(resourceGroup, dName)
		if err != nil {
			return fmt.Errorf("Bad: Get on diskClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound && shouldExist {
			return fmt.Errorf("Bad: ManagedDisk %q (resource group %q) does not exist", dName, resourceGroup)
		}
		if resp.StatusCode != http.StatusNotFound && !shouldExist {
			return fmt.Errorf("Bad: ManagedDisk %q (resource group %q) still exists", dName, resourceGroup)
		}

		*d = resp

		return nil
	}
}

func testCheckAzureRMManagedDiskDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).diskClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_managed_disk" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Managed Disk still exists: \n%#v", resp.Properties)
		}
	}

	return nil
}

func testDeleteAzureRMVirtualMachine(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		vmName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for virtual machine: %s", vmName)
		}

		conn := testAccProvider.Meta().(*ArmClient).vmClient

		_, error := conn.Delete(resourceGroup, vmName, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on vmClient: %s", err)
		}

		return nil
	}
}

var testAccAzureRMManagedDisk_empty = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_managed_disk" "test" {
    name = "acctestd-%d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_type = "Standard_LRS"
    create_option = "Empty"
    disk_size_gb = "1"

    tags {
        environment = "acctest"
        cost-center = "ops"
    }
}`

var testAccAzureRMManagedDisk_import = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_storage_account" "test" {
    name = "accsa%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US 2"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_managed_disk" "test" {
    name = "acctestd-%d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_type = "Standard_LRS"
    create_option = "Import"
    source_uri = "${azurerm_storage_account.test.primary_blob_endpoint}${azurerm_storage_container.test.name}/myosdisk1.vhd"
    disk_size_gb = "45"

    tags {
        environment = "acctest"
    }
}`

var testAccAzureRMManagedDisk_copy = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_managed_disk" "source" {
    name = "acctestd1-%d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_type = "Standard_LRS"
    create_option = "Empty"
    disk_size_gb = "1"

    tags {
        environment = "acctest"
        cost-center = "ops"
    }
}

resource "azurerm_managed_disk" "test" {
    name = "acctestd2-%d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_type = "Standard_LRS"
    create_option = "Copy"
    source_resource_id = "${azurerm_managed_disk.source.id}"
    disk_size_gb = "1"

    tags {
        environment = "acctest"
        cost-center = "ops"
    }
}`

var testAccAzureRMManagedDisk_empty_updated = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_managed_disk" "test" {
    name = "acctestd-%d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_type = "Premium_LRS"
    create_option = "Empty"
    disk_size_gb = "2"

    tags {
        environment = "acctest"
    }
}`

func testAccAzureRMManagedDiskNonStandardCasing(ri int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}
resource "azurerm_managed_disk" "test" {
    name = "acctestd-%d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_type = "standard_lrs"
    create_option = "Empty"
    disk_size_gb = "1"
    tags {
        environment = "acctest"
        cost-center = "ops"
    }
}`, ri, ri)
}
