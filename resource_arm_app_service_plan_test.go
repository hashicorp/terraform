package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/web"
	"github.com/hashicorp/terraform/helper/schema"
)

func TestAccAzureRMAppServicePlan_standard(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(TestAccAzureRMAppServicePlan_standard, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMAppServicePlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAppServicePlanExists("azurerm_app_service_plan.test"),
				),
			},
		},
	})
}

func testCheckAzureRMAppServicePlanDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).appServicePlansClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_app_service_plan" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("App Service Plan still exists:\n%#v", resp)
		}
	}

	return nil
}

func testCheckAzureRMAppServicePlanExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		appServicePlanName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for App Service Plan: %s", redisName)
		}

		conn := testAccProvider.Meta().(*ArmClient).appServicePlansClient

		resp, err := conn.Get(resourceGroup, appServicePlanName)
		if err != nil {
			return fmt.Errorf("Bad: Get on appServicePlansClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: App Service Plan %q (resource group: %q) does not exist", appServicePlanName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMAppServicePlan_standard = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_app_service_plan" "test" {
    name                = "acctestAppServicePlan-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    tier                = "S0"
    tags {
    	environment = "production"
    }
}
`
