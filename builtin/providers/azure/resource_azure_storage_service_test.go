package azure

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management/storageservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureStorageService(t *testing.T) {
	name := "azure_storage_service.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAzureStorageServiceDestroyed,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureStorageServiceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAzureStorageServiceExists(name),
					resource.TestCheckResourceAttr(name, "name", "tftesting"),
					resource.TestCheckResourceAttr(name, "location", "North Europe"),
					resource.TestCheckResourceAttr(name, "description", "very descriptive"),
					resource.TestCheckResourceAttr(name, "account_type", "Standard_LRS"),
				),
			},
		},
	})
}

func testAccAzureStorageServiceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure Storage Service Resource not found: %s", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Storage Service ID not set.")
		}

		mgmtClient := testAccProvider.Meta().(*Client).mgmtClient
		_, err := storageservice.NewClient(mgmtClient).GetStorageService(resource.Primary.ID)

		return err
	}
}

func testAccAzureStorageServiceDestroyed(s *terraform.State) error {
	mgmgClient := testAccProvider.Meta().(*Client).mgmtClient

	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_storage_service" {
			continue
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Storage Service ID not set.")
		}

		_, err := storageservice.NewClient(mgmgClient).GetStorageService(resource.Primary.ID)
		return testAccResourceDestroyedErrorFilter("Storage Service", err)
	}

	return nil
}

var testAccAzureStorageServiceConfig = `
resource "azure_storage_service" "foo" {
    # NOTE: storage service names constrained to lowercase letters only.
	name = "tftesting"
	location = "West US"
    description = "very descriptive"
	account_type = "Standard_LRS"
}
`
