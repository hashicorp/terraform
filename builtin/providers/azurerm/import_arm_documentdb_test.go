package azurerm

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMDocumentDb_importStandardBoundedStaleness(t *testing.T) {
	resourceName := "azurerm_documentdb.test"

	ri := acctest.RandInt()
	config := testAccAzureRMDocumentDb_standard_boundedStaleness(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDocumentDbDestroy,
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

func TestAccAzureRMDocumentDb_importStandardEventualConsistency(t *testing.T) {
	resourceName := "azurerm_documentdb.test"

	ri := acctest.RandInt()
	config := testAccAzureRMDocumentDb_standard_eventualConsistency(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDocumentDbDestroy,
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

func TestAccAzureRMDocumentDb_importStandardGeoReplicated(t *testing.T) {
	resourceName := "azurerm_documentdb.test"

	ri := acctest.RandInt()
	config := testAccAzureRMDocumentDb_standardGeoReplicated(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDocumentDbDestroy,
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
