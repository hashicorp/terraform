package azurerm

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMSubnet_importBasic(t *testing.T) {
	resourceName := "azurerm_subnet.test"

	ri := acctest.RandInt()
	config := testAccAzureRMSubnet_basic(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSubnetDestroy,
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

func TestAccAzureRMSubnet_importWithRouteTable(t *testing.T) {
	resourceName := "azurerm_subnet.test"

	ri := acctest.RandInt()
	config := testAccAzureRMSubnet_routeTable(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSubnetDestroy,
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
