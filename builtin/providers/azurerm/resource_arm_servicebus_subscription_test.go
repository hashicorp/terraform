package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMServiceBusSubscription_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMServiceBusSubscription_basic, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusSubscriptionExists("azurerm_servicebus_subscription.test"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusSubscription_update(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusSubscription_basic, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusSubscription_update, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusSubscriptionExists("azurerm_servicebus_subscription.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_subscription.test", "enable_batched_operations", "true"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusSubscription_updateRequiresSession(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusSubscription_basic, ri, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusSubscription_updateRequiresSession, ri, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusSubscriptionExists("azurerm_servicebus_subscription.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_subscription.test", "requires_session", "true"),
				),
			},
		},
	})
}

func testCheckAzureRMServiceBusSubscriptionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArmClient).serviceBusSubscriptionsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_servicebus_subscription" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		topicName := rs.Primary.Attributes["topic_name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := client.Get(resourceGroup, namespaceName, topicName, name)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return nil
			}
			return err
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("ServiceBus Subscription still exists:\n%+v", resp.SubscriptionProperties)
		}
	}

	return nil
}

func testCheckAzureRMServiceBusSubscriptionExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		subscriptionName := rs.Primary.Attributes["name"]
		topicName := rs.Primary.Attributes["topic_name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for subscription: %s", topicName)
		}

		client := testAccProvider.Meta().(*ArmClient).serviceBusSubscriptionsClient

		resp, err := client.Get(resourceGroup, namespaceName, topicName, subscriptionName)
		if err != nil {
			return fmt.Errorf("Bad: Get on serviceBusSubscriptionsClient: %+v", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Subscription %q (resource group: %q) does not exist", subscriptionName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMServiceBusSubscription_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_servicebus_namespace" "test" {
    name = "acctestservicebusnamespace-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "standard"
}

resource "azurerm_servicebus_topic" "test" {
    name = "acctestservicebustopic-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_servicebus_subscription" "test" {
    name = "acctestservicebussubscription-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    topic_name = "${azurerm_servicebus_topic.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    max_delivery_count = 10
}
`

var testAccAzureRMServiceBusSubscription_update = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_servicebus_namespace" "test" {
    name = "acctestservicebusnamespace-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "standard"
}

resource "azurerm_servicebus_topic" "test" {
    name = "acctestservicebustopic-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_servicebus_subscription" "test" {
    name = "acctestservicebussubscription-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    topic_name = "${azurerm_servicebus_topic.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    max_delivery_count = 10
    enable_batched_operations = true
}
`

var testAccAzureRMServiceBusSubscription_updateRequiresSession = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_servicebus_namespace" "test" {
    name = "acctestservicebusnamespace-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "standard"
}

resource "azurerm_servicebus_topic" "test" {
    name = "acctestservicebustopic-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_servicebus_subscription" "test" {
    name = "acctestservicebussubscription-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    topic_name = "${azurerm_servicebus_topic.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    max_delivery_count = 10
    requires_session = true
}
`
