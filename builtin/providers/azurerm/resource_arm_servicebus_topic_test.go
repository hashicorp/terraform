package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMServiceBusTopic_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMServiceBusTopic_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusTopicExists("azurerm_servicebus_topic.test"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusTopic_update(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_update, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusTopicExists("azurerm_servicebus_topic.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_topic.test", "enable_batched_operations", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_topic.test", "enable_express", "true"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusTopic_enablePartitioningStandard(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_enablePartitioningStandard, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusTopicExists("azurerm_servicebus_topic.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_topic.test", "enable_partitioning", "true"),
					// Ensure size is read back in it's original value and not the x16 value returned by Azure
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_topic.test", "max_size_in_megabytes", "5120"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusTopic_enablePartitioningPremium(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_enablePartitioningPremium, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusTopicExists("azurerm_servicebus_topic.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_topic.test", "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_topic.test", "max_size_in_megabytes", "81920"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusTopic_enableDuplicateDetection(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusTopic_enableDuplicateDetection, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusTopicExists("azurerm_servicebus_topic.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_topic.test", "requires_duplicate_detection", "true"),
				),
			},
		},
	})
}

func testCheckAzureRMServiceBusTopicDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArmClient).serviceBusTopicsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_servicebus_topic" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := client.Get(resourceGroup, namespaceName, name)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return nil
			}
			return err
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("ServiceBus Topic still exists:\n%+v", resp.TopicProperties)
		}
	}

	return nil
}

func testCheckAzureRMServiceBusTopicExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		topicName := rs.Primary.Attributes["name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for topic: %s", topicName)
		}

		client := testAccProvider.Meta().(*ArmClient).serviceBusTopicsClient

		resp, err := client.Get(resourceGroup, namespaceName, topicName)
		if err != nil {
			return fmt.Errorf("Bad: Get on serviceBusTopicsClient: %+v", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Topic %q (resource group: %q) does not exist", namespaceName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMServiceBusTopic_basic = `
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
`

var testAccAzureRMServiceBusTopic_basicPremium = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_servicebus_namespace" "test" {
    name = "acctestservicebusnamespace-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "premium"
}

resource "azurerm_servicebus_topic" "test" {
    name = "acctestservicebustopic-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMServiceBusTopic_update = `
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
    enable_batched_operations = true
    enable_express = true
}
`

var testAccAzureRMServiceBusTopic_enablePartitioningStandard = `
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
    enable_partitioning = true
	max_size_in_megabytes = 5120
}
`

var testAccAzureRMServiceBusTopic_enablePartitioningPremium = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_servicebus_namespace" "test" {
    name = "acctestservicebusnamespace-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "premium"
}

resource "azurerm_servicebus_topic" "test" {
    name = "acctestservicebustopic-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_partitioning = true
	max_size_in_megabytes = 81920
}
`

var testAccAzureRMServiceBusTopic_enableDuplicateDetection = `
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
    requires_duplicate_detection = true
}
`
