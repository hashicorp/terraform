package azure

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureStorageBlockBlob(t *testing.T) {
	name := "azure_storage_blob.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureStorageBlobDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureStorageBlockBlobConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureStorageBlobExists(name),
					resource.TestCheckResourceAttr(name, "name", "tftesting-blob"),
					resource.TestCheckResourceAttr(name, "type", "BlockBlob"),
					resource.TestCheckResourceAttr(name, "storage_container_name", testAccStorageContainerName),
					resource.TestCheckResourceAttr(name, "storage_service_name", testAccStorageServiceName),
				),
			},
		},
	})

	// because containers take a while to get deleted, sleep for a while:
	time.Sleep(5 * time.Minute)
}

func TestAccAzureStoragePageBlob(t *testing.T) {
	name := "azure_storage_blob.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureStorageBlobDeleted,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureStoragePageBlobConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureStorageBlobExists(name),
					resource.TestCheckResourceAttr(name, "name", "tftesting-blob"),
					resource.TestCheckResourceAttr(name, "type", "PageBlob"),
					resource.TestCheckResourceAttr(name, "size", "512"),
					resource.TestCheckResourceAttr(name, "storage_container_name", testAccStorageContainerName),
					resource.TestCheckResourceAttr(name, "storage_service_name", testAccStorageServiceName),
				),
			},
		},
	})

	// because containers take a while to get deleted, sleep for a while:
	time.Sleep(5 * time.Minute)
}

func testAccCheckAzureStorageBlobExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Azure Storage Container resource not found: %s", name)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("Azure Storage Container ID not set: %s", name)
		}

		mgmtClient := testAccProvider.Meta().(*Client).mgmtClient
		blobClient, err := getStorageServiceBlobClient(mgmtClient, testAccStorageServiceName)
		if err != nil {
			return err
		}

		exists, err := blobClient.BlobExists(testAccStorageContainerName, resource.Primary.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Azure Storage Blob %s doesn't exist.", name)
		}

		return nil
	}
}

func testAccCheckAzureStorageBlobDeleted(s *terraform.State) error {
	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azure_storage_blob" {
			continue
		}

		mgmtClient := testAccProvider.Meta().(*Client).mgmtClient
		blobClient, err := getStorageServiceBlobClient(mgmtClient, testAccStorageServiceName)
		if err != nil {
			return err
		}

		exists, err := blobClient.BlobExists(testAccStorageContainerName, resource.Primary.ID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Azure Storage Blob still exists.")
		}
	}

	return nil

}

var testAccAzureStorageBlockBlobConfig = testAccAzureStorageContainerConfig + fmt.Sprintf(`
resource "azure_storage_blob" "foo" {
	name = "tftesting-blob"
	type = "BlockBlob"
    # NOTE: A pre-existing Storage Service is used here so as to avoid
    # the huge wait for creation of one.
	storage_service_name = "%s"
	storage_container_name = "%s"
}
`, testAccStorageServiceName, testAccStorageContainerName)

var testAccAzureStoragePageBlobConfig = testAccAzureStorageContainerConfig + fmt.Sprintf(`
resource "azure_storage_blob" "foo" {
	name = "tftesting-blob"
	type = "PageBlob"
    # NOTE: A pre-existing Storage Service is used here so as to avoid
    # the huge wait for creation of one.
	storage_service_name = "%s"
	storage_container_name = "%s"
    # NOTE: must be a multiple of 512:
    size = 512
}
`, testAccStorageServiceName, testAccStorageContainerName)
