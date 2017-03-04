package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMManagedDisk_importEmpty(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMManagedDisk_empty, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMManagedDiskDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      "azurerm_managed_disk.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
