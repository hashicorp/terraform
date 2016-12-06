package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMApplicationInsights_importWeb(t *testing.T) {
	resourceName := "azurerm_application_insights.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_web, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
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

func TestAccAzureRMApplicationInsights_importWebWithTags(t *testing.T) {
	resourceName := "azurerm_application_insights.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_webWithTags, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
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

func TestAccAzureRMApplicationInsights_importOther(t *testing.T) {
	resourceName := "azurerm_application_insights.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_other, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
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

func TestAccAzureRMApplicationInsights_importOtherWithTags(t *testing.T) {
	resourceName := "azurerm_application_insights.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_otherWithTags, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
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
