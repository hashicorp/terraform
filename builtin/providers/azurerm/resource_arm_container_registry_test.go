package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMContainerRegistryName_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "four",
			ErrCount: 1,
		},
		{
			Value:    "5five",
			ErrCount: 0,
		},
		{
			Value:    "hello-world",
			ErrCount: 1,
		},
		{
			Value:    "hello_world",
			ErrCount: 1,
		},
		{
			Value:    "helloWorld",
			ErrCount: 0,
		},
		{
			Value:    "helloworld12",
			ErrCount: 0,
		},
		{
			Value:    "hello@world",
			ErrCount: 1,
		},
		{
			Value:    "qfvbdsbvipqdbwsbddbdcwqffewsqwcdw21ddwqwd3324120",
			ErrCount: 0,
		},
		{
			Value:    "qfvbdsbvipqdbwsbddbdcwqffewsqwcdw21ddwqwd33241202",
			ErrCount: 0,
		},
		{
			Value:    "qfvbdsbvipqdbwsbddbdcwqfjjfewsqwcdw21ddwqwd3324120",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAzureRMContainerRegistryName(tc.Value, "azurerm_container_registry")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Container Registry Name to trigger a validation error: %v", errors)
		}
	}
}

func TestAccAzureRMContainerRegistry_basic(t *testing.T) {
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
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMContainerRegistryExists("azurerm_container_registry.test"),
				),
			},
		},
	})
}

func TestAccAzureRMContainerRegistry_complete(t *testing.T) {
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
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMContainerRegistryExists("azurerm_container_registry.test"),
				),
			},
		},
	})
}

func TestAccAzureRMContainerRegistry_update(t *testing.T) {
	ri := acctest.RandInt()
	rs := acctest.RandString(4)
	config := testAccAzureRMContainerRegistry_complete(ri, rs)
	updatedConfig := testAccAzureRMContainerRegistry_completeUpdated(ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMContainerRegistryDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMContainerRegistryExists("azurerm_container_registry.test"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMContainerRegistryExists("azurerm_container_registry.test"),
				),
			},
		},
	})
}

func testCheckAzureRMContainerRegistryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).containerRegistryClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_container_registry" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Container Registry still exists:\n%#v", resp)
		}
	}

	return nil
}

func testCheckAzureRMContainerRegistryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Container Registry: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).containerRegistryClient

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on containerRegistryClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Container Registry %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testAccAzureRMContainerRegistry_basic(rInt int, rStr string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "testAccRg-%d"
  location = "West US"
}

resource "azurerm_storage_account" "test" {
  name                = "testaccsa%s"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  account_type        = "Standard_LRS"
}

resource "azurerm_container_registry" "test" {
  name                = "testacccr%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  sku                 = "Basic"

  storage_account {
    name       = "${azurerm_storage_account.test.name}"
    access_key = "${azurerm_storage_account.test.primary_access_key}"
  }
}
`, rInt, rStr, rInt)
}

func testAccAzureRMContainerRegistry_complete(rInt int, rStr string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "testAccRg-%d"
  location = "West US"
}

resource "azurerm_storage_account" "test" {
  name                = "testaccsa%s"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  account_type        = "Standard_LRS"
}

resource "azurerm_container_registry" "test" {
  name                = "testacccr%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  admin_enabled       = false
  sku                 = "Basic"

  storage_account {
    name       = "${azurerm_storage_account.test.name}"
    access_key = "${azurerm_storage_account.test.primary_access_key}"
  }

  tags {
    environment = "production"
  }
}
`, rInt, rStr, rInt)
}

func testAccAzureRMContainerRegistry_completeUpdated(rInt int, rStr string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "testAccRg-%d"
  location = "West US"
}

resource "azurerm_storage_account" "test" {
  name                = "testaccsa%s"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  account_type        = "Standard_LRS"
}

resource "azurerm_container_registry" "test" {
  name                = "testacccr%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  admin_enabled       = true
  sku                 = "Basic"

  storage_account {
    name       = "${azurerm_storage_account.test.name}"
    access_key = "${azurerm_storage_account.test.primary_access_key}"
  }

  tags {
    environment = "production"
  }
}
`, rInt, rStr, rInt)
}
