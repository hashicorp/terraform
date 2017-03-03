package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMDisk_importEmpty(t *testing.T) {
	runTestAzureRMDisk_import(t, "azurerm_disk.test", testAccAzureRMDisk_emptyDisk)
}

func TestAccAzureRMDisk_importBlob(t *testing.T) {
	runTestAzureRMDisk_import(t, "azurerm_disk.test", testAccAzureRMDisk_blob)
}

func runTestAzureRMDisk_import(t *testing.T, resourceName string, configSource string) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(configSource, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t)},
		Providers: testAccProviders,
		CheckDestroy: testCheckAzureRMDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
			},

			resource.TestStep{
				ResourceName: resourceName,
				ImportState: true,
				ImportStateVerify: true,
			},
		},
	})
}