package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMServiceBusNamespaceCapacity_validation(t *testing.T) {
	cases := []struct {
		Value    int
		ErrCount int
	}{
		{
			Value:    17,
			ErrCount: 1,
		},
		{
			Value:    1,
			ErrCount: 0,
		},
		{
			Value:    2,
			ErrCount: 0,
		},
		{
			Value:    4,
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateServiceBusNamespaceCapacity(tc.Value, "azurerm_servicebus_namespace")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM ServiceBus Namespace Capacity to trigger a validation error")
		}
	}
}

func TestAccAzureRMServiceBusNamespaceSku_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Basic",
			ErrCount: 0,
		},
		{
			Value:    "Standard",
			ErrCount: 0,
		},
		{
			Value:    "Premium",
			ErrCount: 0,
		},
		{
			Value:    "Random",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateServiceBusNamespaceSku(tc.Value, "azurerm_servicebus_namespace")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM ServiceBus Namespace Sku to trigger a validation error")
		}
	}
}

func TestAccAzureRMServiceBusNamespace_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMServiceBusNamespace_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusNamespaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusNamespaceExists("azurerm_servicebus_namespace.test"),
				),
			},
		},
	})
}

func testCheckAzureRMServiceBusNamespaceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).serviceBusNamespacesClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_servicebus_namespace" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("ServiceBus Namespace still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

func testCheckAzureRMServiceBusNamespaceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		namespaceName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for public ip: %s", namespaceName)
		}

		conn := testAccProvider.Meta().(*ArmClient).serviceBusNamespacesClient

		resp, err := conn.Get(resourceGroup, namespaceName)
		if err != nil {
			return fmt.Errorf("Bad: Get on serviceBusNamespacesClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Public IP %q (resource group: %q) does not exist", namespaceName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMServiceBusNamespace_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}
resource "azurerm_servicebus_namespace" "test" {
    name = "acctestservicebusnamespace-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "basic"
}
`
