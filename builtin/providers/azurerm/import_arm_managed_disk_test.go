package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMManagedDisk_importEmpty(t *testing.T) {
	runTestAzureRMManagedDisk_import(t, "azurerm_disk.test", testAccAzureRMManagedDisk_empty)
}

/*func TestAccAzureRMManagedDisk_importBlob(t *testing.T) {
	runTestAzureRMManagedDisk_import(t, "azurerm_disk.test", testAccAzureRMManagedDisk_blob)
}*/

func runTestAzureRMManagedDisk_import(t *testing.T, resourceName string, configSource string) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(configSource, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
