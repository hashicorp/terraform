package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMEventHubConsumerGroup_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubConsumerGroup_basic, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubConsumerGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubConsumerGroupExists("azurerm_eventhub_consumer_group.test"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubConsumerGroup_complete(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMEventHubConsumerGroup_complete, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMEventHubConsumerGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubConsumerGroupExists("azurerm_eventhub_consumer_group.test"),
				),
			},
		},
	})
}

func testCheckAzureRMEventHubConsumerGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).eventHubConsumerGroupClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_eventhub_consumer_group" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		eventHubName := rs.Primary.Attributes["eventhub_name"]

		resp, err := conn.Get(resourceGroup, namespaceName, eventHubName, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("EventHub Consumer Group still exists:\n%#v", resp.ConsumerGroupProperties)
		}
	}

	return nil
}

func testCheckAzureRMEventHubConsumerGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Event Hub Consumer Group: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).eventHubConsumerGroupClient

		namespaceName := rs.Primary.Attributes["namespace_name"]
		eventHubName := rs.Primary.Attributes["eventhub_name"]

		resp, err := conn.Get(resourceGroup, namespaceName, eventHubName, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on eventHubConsumerGroupClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Event Hub Consumer Group %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMEventHubConsumerGroup_basic = `
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
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    partition_count     = 2
    message_retention   = 7
}

resource "azurerm_eventhub_consumer_group" "test" {
    name = "acctesteventhubcg-%d"
    namespace_name      = "${azurerm_eventhub_namespace.test.name}"
    eventhub_name       = "${azurerm_eventhub.test.name}"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMEventHubConsumerGroup_complete = `
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

resource "azurerm_eventhub_consumer_group" "test" {
    name                = "acctesteventhubcg-%d"
    namespace_name      = "${azurerm_eventhub_namespace.test.name}"
    eventhub_name       = "${azurerm_eventhub.test.name}"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    user_metadata       = "some-meta-data"
}
`
