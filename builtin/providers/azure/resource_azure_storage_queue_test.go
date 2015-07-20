package azure

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureStorageQueue(t *testing.T) {
	name := "azure_storage_queue.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureStorageQueueDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureStorageQueueConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureStorageQueueExists(name),
					resource.TestCheckResourceAttr(name, "name", "terraform-queue"),
					resource.TestCheckResourceAttr(name, "storage_service_name", testAccStorageServiceName),
				),
			},
		},
	})
}

func testAccCheckAzureStorageQueueExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure Storage Queue resource '%s' is missing.", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Storage Service Queue ID %s is missing.", name)
		}

		azureClient := testAccProvider.Meta().(*Client)
		queueClient, err := azureClient.getStorageServiceQueueClient(testAccStorageServiceName)
		if err != nil {
			return err
		}

		exists, err := queueClient.QueueExists(resource.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error querying Azure for Storage Queue existence: %s", err)
		}
		if !exists {
			return fmt.Errorf("Azure Storage Queue %s doesn't exist!", resource.Primary.ID)
		}

		return nil
	}
}

func testAccCheckAzureStorageQueueDeleted(s *terraform.State) error {
	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_storage_queue" {
			continue
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Storage Service Queue ID %s is missing.", resource.Primary.ID)
		}

		azureClient := testAccProvider.Meta().(*Client)
		queueClient, err := azureClient.getStorageServiceQueueClient(testAccStorageServiceName)
		if err != nil {
			return err
		}

		exists, err := queueClient.QueueExists(resource.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error querying Azure for Storage Queue existence: %s", err)
		}
		if exists {
			return fmt.Errorf("Azure Storage Queue %s still exists!", resource.Primary.ID)
		}
	}

	return nil
}

var testAccAzureStorageQueueConfig = fmt.Sprintf(`
resource "azure_storage_queue" "foo" {
    name = "terraform-queue"
    storage_service_name = "%s"
}
`, testAccStorageServiceName)
