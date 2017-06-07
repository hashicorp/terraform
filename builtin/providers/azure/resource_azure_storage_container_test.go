package azure

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureStorageContainer(t *testing.T) {
	name := "azure_storage_container.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureStorageContainerDestroyed,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureStorageContainerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureStorageContainerExists(name),
					resource.TestCheckResourceAttr(name, "name", testAccStorageContainerName),
					resource.TestCheckResourceAttr(name, "storage_service_name", testAccStorageServiceName),
					resource.TestCheckResourceAttr(name, "container_access_type", "blob"),
				),
			},
		},
	})

	// because containers take a while to get deleted, sleep for one minute:
	time.Sleep(3 * time.Minute)
}

func testAccCheckAzureStorageContainerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure Storage Container resource not found: %s", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Storage Container ID not set: %s", name)
		}

		azureClient := testAccProvider.Meta().(*Client)
		blobClient, err := azureClient.getStorageServiceBlobClient(testAccStorageServiceName)
		if err != nil {
			return err
		}

		container := blobClient.GetContainerReference(resource.Primary.ID)
		exists, err := container.Exists()
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Azure Storage Container %s doesn't exist.", name)
		}

		return nil
	}
}

func testAccCheckAzureStorageContainerDestroyed(s *terraform.State) error {
	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_storage_container" {
			continue
		}

		azureClient := testAccProvider.Meta().(*Client)
		blobClient, err := azureClient.getStorageServiceBlobClient(testAccStorageServiceName)
		if err != nil {
			return err
		}

		container := blobClient.GetContainerReference(resource.Primary.ID)
		exists, err := container.Exists()
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Azure Storage Container still exists.")
		}
	}

	return nil
}

var testAccAzureStorageContainerConfig = fmt.Sprintf(`
resource "azure_storage_container" "foo" {
	name = "%s"
	container_access_type = "blob"
    # NOTE: A pre-existing Storage Service is used here so as to avoid
    # the huge wait for creation of one.
	storage_service_name = "%s"
}
`, testAccStorageContainerName, testAccStorageServiceName)
