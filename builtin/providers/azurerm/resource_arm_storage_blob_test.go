package azurerm

import (
	"fmt"
	"io/ioutil"
	"testing"

	"strings"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMStorageBlobType_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "unknown",
			ErrCount: 1,
		},
		{
			Value:    "page",
			ErrCount: 0,
		},
		{
			Value:    "block",
			ErrCount: 0,
		},
		{
			Value:    "BLOCK",
			ErrCount: 0,
		},
		{
			Value:    "Block",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmStorageBlobType(tc.Value, "azurerm_storage_blob")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Storage Blob type to trigger a validation error")
		}
	}
}

func TestResourceAzureRMStorageBlobSize_validation(t *testing.T) {
	cases := []struct {
		Value    int
		ErrCount int
	}{
		{
			Value:    511,
			ErrCount: 1,
		},
		{
			Value:    512,
			ErrCount: 0,
		},
		{
			Value:    1024,
			ErrCount: 0,
		},
		{
			Value:    2048,
			ErrCount: 0,
		},
		{
			Value:    5120,
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmStorageBlobSize(tc.Value, "azurerm_storage_blob")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Storage Blob size to trigger a validation error")
		}
	}
}

func TestAccAzureRMStorageBlob_basic(t *testing.T) {
	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := fmt.Sprintf(testAccAzureRMStorageBlob_basic, ri, rs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMStorageBlobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageBlobExists("azurerm_storage_blob.test"),
				),
			},
		},
	})
}

func TestAccAzureRMStorageBlob_remote(t *testing.T) {
	ri := acctest.RandInt()
	rs1 := strings.ToLower(acctest.RandString(11))
	rs2 := strings.ToLower(acctest.RandString(11))
	sourceBlob, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Failed to create local source blob file")
	}

	_, err = sourceBlob.WriteString(rs1)
	if err != nil {
		t.Fatalf("Failed to write random test to source blob")
	}

	err = sourceBlob.Close()
	if err != nil {
		t.Fatalf("Failed to close source blob")
	}

	config := fmt.Sprintf(testAccAzureRMStorageBlob_remote, ri, rs1, sourceBlob.Name(), rs2)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		//CheckDestroy: testCheckAzureRMStorageBlobDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMStorageBlobMatches("azurerm_storage_blob.destination", rs1),
				),
			},
		},
	})
}

func testCheckAzureRMStorageBlobExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		storageContainerName := rs.Primary.Attributes["storage_container_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage blob: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			return fmt.Errorf("Bad: Storage Account %q does not exist", storageAccountName)
		}

		exists, err := blobClient.BlobExists(storageContainerName, name)
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf("Bad: Storage Blob %q (storage container: %q) does not exist", name, storageContainerName)
		}

		return nil
	}
}

func testCheckAzureRMStorageBlobMatches(name, expectedContents string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		storageContainerName := rs.Primary.Attributes["storage_container_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage blob: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			return fmt.Errorf("Bad: Storage Account %q does not exist", storageAccountName)
		}

		blob, err := blobClient.GetBlob(storageContainerName, name)
		if err != nil {
			return err
		}

		contents, err := ioutil.ReadAll(blob)
		if err != nil {
			return err
		}
		defer blob.Close()

		if string(contents) != expectedContents {
			return fmt.Errorf("Bad: Storage Blob %q (storage container: %q) does not match contents %q (found: %q)", name, storageContainerName, expectedContents, contents)
		}

		return nil
	}
}

func testCheckAzureRMStorageBlobDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_storage_blob" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		storageContainerName := rs.Primary.Attributes["storage_container_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage blob: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(resourceGroup, storageAccountName)
		if err != nil {
			return nil
		}
		if !accountExists {
			return nil
		}

		exists, err := blobClient.BlobExists(storageContainerName, name)
		if err != nil {
			return nil
		}

		if exists {
			return fmt.Errorf("Bad: Storage Blob %q (storage container: %q) still exists", name, storageContainerName)
		}
	}

	return nil
}

var testAccAzureRMStorageBlob_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "westus"
}

resource "azurerm_storage_account" "test" {
    name = "acctestacc%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_storage_blob" "test" {
    name = "herpderp1.vhd"

    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    storage_container_name = "${azurerm_storage_container.test.name}"

    type = "page"
    size = 5120
}
`

var testAccAzureRMStorageBlob_remote = `
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "westus"
}

resource "azurerm_storage_account" "source" {
    name = "acctestacc%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "source" {
    name = "source"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.source.name}"
    container_access_type = "blob"
}

resource "azurerm_storage_blob" "source" {
    name = "source.vhd"

    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.source.name}"
    storage_container_name = "${azurerm_storage_container.source.name}"

    type = "block"
		file_path = "%s"
}

resource "azurerm_storage_account" "destination" {
    name = "acctestacc%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"

    tags {
        environment = "staging"
    }
}

resource "azurerm_storage_container" "destination" {
    name = "destination"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.destination.name}"
    container_access_type = "private"
}

resource "azurerm_storage_blob" "destination" {
    name = "destination.vhd"
		depends_on = ["azurerm_storage_blob.source"]

    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.destination.name}"
    storage_container_name = "${azurerm_storage_container.destination.name}"

    type = "block"
		source_blob_url = "${azurerm_storage_blob.source.url}"
}
`
