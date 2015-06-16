package azure

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureStorageBlockBlob(t *testing.T) {
	name := "azure_storage_blob.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureStorageBlobDeleted("block"),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureStorageBlockBlobConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureStorageBlobExists(name, "block"),
					resource.TestCheckResourceAttr(name, "name", "tftesting-blob"),
					resource.TestCheckResourceAttr(name, "type", "BlockBlob"),
					resource.TestCheckResourceAttr(name, "storage_container_name",
						fmt.Sprintf("%s-block", testAccStorageContainerName)),
					resource.TestCheckResourceAttr(name, "storage_service_name", testAccStorageServiceName),
				),
			},
		},
	})
}

func TestAccAzureStoragePageBlob(t *testing.T) {
	name := "azure_storage_blob.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureStorageBlobDeleted("page"),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureStoragePageBlobConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureStorageBlobExists(name, "page"),
					resource.TestCheckResourceAttr(name, "name", "tftesting-blob"),
					resource.TestCheckResourceAttr(name, "type", "PageBlob"),
					resource.TestCheckResourceAttr(name, "size", "512"),
					resource.TestCheckResourceAttr(name, "storage_container_name",
						fmt.Sprintf("%s-page", testAccStorageContainerName)),
					resource.TestCheckResourceAttr(name, "storage_service_name", testAccStorageServiceName),
				),
			},
		},
	})
}

func testAccCheckAzureStorageBlobExists(name, typ string) resource.TestCheckFunc {
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

		exists, err := blobClient.BlobExists(fmt.Sprintf("%s-%s", testAccStorageContainerName, typ),
			resource.Primary.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Azure Storage Blob %s doesn't exist.", name)
		}

		return nil
	}
}

func testAccCheckAzureStorageBlobDeleted(typ string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, resource := range s.RootModule().Resources {
			if resource.Type != "azure_storage_blob" {
				continue
			}

			azureClient := testAccProvider.Meta().(*Client)
			blobClient, err := azureClient.getStorageServiceBlobClient(testAccStorageServiceName)
			if err != nil {
				return err
			}

			exists, err := blobClient.BlobExists(fmt.Sprintf("%s-%s", testAccStorageContainerName,
				typ), resource.Primary.ID)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("Azure Storage Blob still exists.")
			}
		}

		return nil
	}
}

var testAccAzureStorageBlockBlobConfig = fmt.Sprintf(`
resource "azure_storage_container" "foo" {
	name = "%s-block"
	container_access_type = "blob"
    # NOTE: A pre-existing Storage Service is used here so as to avoid
    # the huge wait for creation of one.
	storage_service_name = "%s"
}

resource "azure_storage_blob" "foo" {
	name = "tftesting-blob"
	type = "BlockBlob"
    # NOTE: A pre-existing Storage Service is used here so as to avoid
    # the huge wait for creation of one.
	storage_service_name = "${azure_storage_container.foo.storage_service_name}"
	storage_container_name = "${azure_storage_container.foo.name}"
}
`, testAccStorageContainerName, testAccStorageServiceName)

var testAccAzureStoragePageBlobConfig = fmt.Sprintf(`
resource "azure_storage_container" "foo" {
	name = "%s-page"
	container_access_type = "blob"
    # NOTE: A pre-existing Storage Service is used here so as to avoid
    # the huge wait for creation of one.
	storage_service_name = "%s"
}

resource "azure_storage_blob" "foo" {
	name = "tftesting-blob"
	type = "PageBlob"
    # NOTE: A pre-existing Storage Service is used here so as to avoid
    # the huge wait for creation of one.
	storage_service_name = "${azure_storage_container.foo.storage_service_name}"
	storage_container_name = "${azure_storage_container.foo.name}"
    # NOTE: must be a multiple of 512:
    size = 512
}
`, testAccStorageContainerName, testAccStorageServiceName)
