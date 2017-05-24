package azurerm

import (
	"testing"

	"fmt"
	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAzureRMEventHubNamespace_importBasic(t *testing.T) {
	resourceName := "azurerm_eventhub_namespace.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubNamespace_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDestroy,
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
