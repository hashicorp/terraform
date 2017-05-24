package azurerm

import (
	"testing"

	"fmt"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAzureRMEventHub_importBasic(t *testing.T) {
	resourceName := "azurerm_eventhub.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHub_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubDestroy,
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
