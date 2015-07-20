package azure

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/virtualmachinedisk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureDataDisk_basic(t *testing.T) {
	var disk virtualmachinedisk.DataDiskResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureDataDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureDataDisk_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureDataDiskExists(
						"azure_data_disk.foo", &disk),
					testAccCheckAzureDataDiskAttributes(&disk),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "label", "terraform-test-0"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "size", "10"),
				),
			},
		},
	})
}

func TestAccAzureDataDisk_update(t *testing.T) {
	var disk virtualmachinedisk.DataDiskResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAzureDataDiskDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureDataDisk_advanced,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureDataDiskExists(
						"azure_data_disk.foo", &disk),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "label", "terraform-test1-1"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "lun", "1"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "size", "10"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "caching", "ReadOnly"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "virtual_machine", "terraform-test1"),
				),
			},

			resource.TestStep{
				Config: testAccAzureDataDisk_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureDataDiskExists(
						"azure_data_disk.foo", &disk),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "label", "terraform-test1-1"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "lun", "2"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "size", "20"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "caching", "ReadWrite"),
					resource.TestCheckResourceAttr(
						"azure_data_disk.foo", "virtual_machine", "terraform-test2"),
				),
			},
		},
	})
}

func testAccCheckAzureDataDiskExists(
	n string,
	disk *virtualmachinedisk.DataDiskResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Data Disk ID is set")
		}

		vm := rs.Primary.Attributes["virtual_machine"]
		lun, err := strconv.Atoi(rs.Primary.Attributes["lun"])
		if err != nil {
			return err
		}

		vmDiskClient := testAccProvider.Meta().(*Client).vmDiskClient
		d, err := vmDiskClient.GetDataDisk(vm, vm, vm, lun)
		if err != nil {
			return err
		}

		if d.DiskName != rs.Primary.ID {
			return fmt.Errorf("Data Disk not found")
		}

		*disk = d

		return nil
	}
}

func testAccCheckAzureDataDiskAttributes(
	disk *virtualmachinedisk.DataDiskResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if disk.Lun != 0 {
			return fmt.Errorf("Bad lun: %d", disk.Lun)
		}

		if disk.LogicalDiskSizeInGB != 10 {
			return fmt.Errorf("Bad size: %d", disk.LogicalDiskSizeInGB)
		}

		if disk.HostCaching != "None" {
			return fmt.Errorf("Bad caching: %s", disk.HostCaching)
		}

		return nil
	}
}

func testAccCheckAzureDataDiskDestroy(s *terraform.State) error {
	vmDiskClient := testAccProvider.Meta().(*Client).vmDiskClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azure_data_disk" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Disk ID is set")
		}

		vm := rs.Primary.Attributes["virtual_machine"]
		lun, err := strconv.Atoi(rs.Primary.Attributes["lun"])
		if err != nil {
			return err
		}

		_, err = vmDiskClient.GetDataDisk(vm, vm, vm, lun)
		if err == nil {
			return fmt.Errorf("Data disk %s still exists", rs.Primary.ID)
		}

		if !management.IsResourceNotFoundError(err) {
			return err
		}
	}

	return nil
}

var testAccAzureDataDisk_basic = fmt.Sprintf(`
resource "azure_instance" "foo" {
    name = "terraform-test"
    image = "Ubuntu Server 14.04 LTS"
    size = "Basic_A1"
    storage_service_name = "%s"
    location = "West US"
    username = "terraform"
    password = "Pass!admin123"
}

resource "azure_data_disk" "foo" {
    lun = 0
    size = 10
    storage_service_name = "${azure_instance.foo.storage_service_name}"
    virtual_machine = "${azure_instance.foo.id}"
}`, testAccStorageServiceName)

var testAccAzureDataDisk_advanced = fmt.Sprintf(`
resource "azure_instance" "foo" {
    name = "terraform-test1"
    image = "Ubuntu Server 14.04 LTS"
    size = "Basic_A1"
    storage_service_name = "%s"
    location = "West US"
    username = "terraform"
    password = "Pass!admin123"
}

resource "azure_data_disk" "foo" {
    lun = 1
    size = 10
    caching = "ReadOnly"
    storage_service_name = "${azure_instance.foo.storage_service_name}"
    virtual_machine = "${azure_instance.foo.id}"
}`, testAccStorageServiceName)

var testAccAzureDataDisk_update = fmt.Sprintf(`
resource "azure_instance" "foo" {
    name = "terraform-test1"
    image = "Ubuntu Server 14.04 LTS"
    size = "Basic_A1"
    storage_service_name = "%s"
    location = "West US"
    username = "terraform"
    password = "Pass!admin123"
}

resource "azure_instance" "bar" {
    name = "terraform-test2"
    image = "Ubuntu Server 14.04 LTS"
    size = "Basic_A1"
    storage_service_name = "${azure_instance.foo.storage_service_name}"
    location = "West US"
    username = "terraform"
    password = "Pass!admin123"
}

resource "azure_data_disk" "foo" {
    lun = 2
    size = 20
    caching = "ReadWrite"
    storage_service_name = "${azure_instance.bar.storage_service_name}"
    virtual_machine = "${azure_instance.bar.id}"
}`, testAccStorageServiceName)
