package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/disk"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMDisk_empty(t *testing.T) {
	var d disk.Model
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDisk_empty, ri, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDiskDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDiskExists("azurerm_disk.test", &d),
				),
			},
		},
	})
}

func testCheckAzureRMDiskExists(name string, d *disk.Model) resource.TestCheckFunc {
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

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: VirtualMachine %q (resource group %q) does not exist", dName, resourceGroup)
		}

		*d = resp

		return nil
	}
}

func testCheckAzureRMDiskDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).diskClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_disk" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Disk still exists: \n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMDisk_empty = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US 2"
}

resource "azurerm_disk" "test" {
    name = "acctestd-%d"
    location = "West US 2"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_type = "Standard_LRS"
    create_option = "Empty"
    disk_size_gb = "20"

    tags {
        environment = "acctest"
    }
}`
