package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMEventHubAuthorizationRule_listen(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_listen, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubAuthorizationRuleExists("azurerm_eventhub_authorization_rule.test"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubAuthorizationRule_send(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_send, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubAuthorizationRuleExists("azurerm_eventhub_authorization_rule.test"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubAuthorizationRule_readwrite(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_readwrite, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubAuthorizationRuleExists("azurerm_eventhub_authorization_rule.test"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubAuthorizationRule_manage(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubAuthorizationRule_manage, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubAuthorizationRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubAuthorizationRuleExists("azurerm_eventhub_authorization_rule.test"),
				),
			},
		},
	})
}

func testCheckAzureRMEventHubAuthorizationRuleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).eventHubClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_eventhub_authorization_rule" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		eventHubName := rs.Primary.Attributes["eventhub_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.GetAuthorizationRule(resourceGroup, namespaceName, eventHubName, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("EventHub Authorization Rule still exists:\n%#v", resp)
		}
	}

	return nil
}

func testCheckAzureRMEventHubAuthorizationRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		eventHubName := rs.Primary.Attributes["eventhub_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Event Hub: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).eventHubClient
		resp, err := conn.GetAuthorizationRule(resourceGroup, namespaceName, eventHubName, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on eventHubClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Event Hub Authorization Rule %q (eventhub %s, namespace %s / resource group: %s) does not exist", name, eventHubName, namespaceName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMEventHubAuthorizationRule_listen = `
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "West US"
}
resource "azurerm_eventhub_namespace" "test" {
  name = "acctesteventhubnamespace-%d"
  location = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku = "Standard"
}
resource "azurerm_eventhub" "test" {
  name                = "acctesteventhub-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  partition_count     = 2
  message_retention   = 7
}
resource "azurerm_eventhub_authorization_rule" "test" {
  name                = "acctesteventhubrule-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  eventhub_name       = "${azurerm_eventhub.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  listen              = true
  send                = false
  manage              = false
}`

var testAccAzureRMEventHubAuthorizationRule_send = `
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "West US"
}
resource "azurerm_eventhub_namespace" "test" {
  name = "acctesteventhubnamespace-%d"
  location = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku = "Standard"
}
resource "azurerm_eventhub" "test" {
  name                = "acctesteventhub-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  partition_count     = 2
  message_retention   = 7
}
resource "azurerm_eventhub_authorization_rule" "test" {
  name                = "acctesteventhubrule-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  eventhub_name       = "${azurerm_eventhub.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  listen              = false
  send                = true
  manage              = false
}`

var testAccAzureRMEventHubAuthorizationRule_readwrite = `
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "West US"
}
resource "azurerm_eventhub_namespace" "test" {
  name = "acctesteventhubnamespace-%d"
  location = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku = "Standard"
}
resource "azurerm_eventhub" "test" {
  name                = "acctesteventhub-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  partition_count     = 2
  message_retention   = 7
}
resource "azurerm_eventhub_authorization_rule" "test" {
  name                = "acctesteventhubrule-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  eventhub_name       = "${azurerm_eventhub.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  listen              = true
  send                = true
  manage              = false
}`

var testAccAzureRMEventHubAuthorizationRule_manage = `
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "West US"
}
resource "azurerm_eventhub_namespace" "test" {
  name = "acctesteventhubnamespace-%d"
  location = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku = "Standard"
}
resource "azurerm_eventhub" "test" {
  name                = "acctesteventhub-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  partition_count     = 2
  message_retention   = 7
}
resource "azurerm_eventhub_authorization_rule" "test" {
  name                = "acctesteventhubrule-%d"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  eventhub_name       = "${azurerm_eventhub.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  listen              = true
  send                = true
  manage              = true
}`
