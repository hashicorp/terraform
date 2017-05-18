package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMServiceBusQueue_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMServiceBusQueue_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusQueueExists("azurerm_servicebus_queue.test"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusQueue_update(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_update, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusQueueExists("azurerm_servicebus_queue.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_queue.test", "enable_batched_operations", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_queue.test", "enable_express", "true"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusQueue_enablePartitioningStandard(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_enablePartitioningStandard, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusQueueExists("azurerm_servicebus_queue.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_queue.test", "enable_partitioning", "true"),
					// Ensure size is read back in it's original value and not the x16 value returned by Azure
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_queue.test", "max_size_in_megabytes", "5120"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusQueue_enablePartitioningPremium(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_enablePartitioningPremium, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusQueueExists("azurerm_servicebus_queue.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_queue.test", "enable_partitioning", "true"),
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_queue.test", "max_size_in_megabytes", "81920"),
				),
			},
		},
	})
}

func TestAccAzureRMServiceBusQueue_enableDuplicateDetection(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_basic, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMServiceBusQueue_enableDuplicateDetection, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMServiceBusQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMServiceBusQueueExists("azurerm_servicebus_queue.test"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"azurerm_servicebus_queue.test", "requires_duplicate_detection", "true"),
				),
			},
		},
	})
}

func testCheckAzureRMServiceBusQueueDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArmClient).serviceBusQueuesClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_servicebus_queue" {
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
			return fmt.Errorf("ServiceBus Queue still exists:\n%#v", resp.QueueProperties)
		}
	}

	return nil
}

func testCheckAzureRMServiceBusQueueExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		queueName := rs.Primary.Attributes["name"]
		namespaceName := rs.Primary.Attributes["namespace_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for queue: %s", queueName)
		}

		client := testAccProvider.Meta().(*ArmClient).serviceBusQueuesClient

		resp, err := client.Get(resourceGroup, namespaceName, queueName)
		if err != nil {
			return fmt.Errorf("Bad: Get on serviceBusQueuesClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Queue %q (resource group: %q) does not exist", namespaceName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMServiceBusQueue_basic = `
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

resource "azurerm_servicebus_queue" "test" {
    name = "acctestservicebusqueue-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMServiceBusQueue_basicPremium = `
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

resource "azurerm_servicebus_queue" "test" {
    name = "acctestservicebusqueue-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
}
`

var testAccAzureRMServiceBusQueue_update = `
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

resource "azurerm_servicebus_queue" "test" {
    name = "acctestservicebusqueue-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_batched_operations = true
    enable_express = true
}
`

var testAccAzureRMServiceBusQueue_enablePartitioningStandard = `
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

resource "azurerm_servicebus_queue" "test" {
    name = "acctestservicebusqueue-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_partitioning = true
	max_size_in_megabytes = 5120
}
`

var testAccAzureRMServiceBusQueue_enablePartitioningPremium = `
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

resource "azurerm_servicebus_queue" "test" {
    name = "acctestservicebusqueue-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    enable_partitioning = true
	max_size_in_megabytes = 81920
}
`

var testAccAzureRMServiceBusQueue_enableDuplicateDetection = `
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

resource "azurerm_servicebus_queue" "test" {
    name = "acctestservicebusqueue-%d"
    location = "West US"
    namespace_name = "${azurerm_servicebus_namespace.test.name}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    requires_duplicate_detection = true
}
`
