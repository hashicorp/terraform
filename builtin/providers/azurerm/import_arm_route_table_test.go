package azurerm

import (
	"fmt"
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAzureRMRouteTable_importBasic(t *testing.T) {
	resourceName := "azurerm_route_table.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRouteTable_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRouteTableDestroy,
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
