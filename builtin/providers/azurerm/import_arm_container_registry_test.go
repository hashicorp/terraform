package azurerm

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMContainerRegistry_importBasic(t *testing.T) {
	resourceName := "azurerm_container_registry.test"

	ri := acctest.RandInt()
	rs := acctest.RandString(4)
	config := testAccAzureRMContainerRegistry_basic(ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMContainerRegistryDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"storage_account"},
			},
		},
	})
}

func TestAccAzureRMContainerRegistry_importComplete(t *testing.T) {
	resourceName := "azurerm_container_registry.test"

	ri := acctest.RandInt()
	rs := acctest.RandString(4)
	config := testAccAzureRMContainerRegistry_complete(ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMContainerRegistryDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"storage_account"},
			},
		},
	})
}
