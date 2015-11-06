package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/blockstorage/v1/volumes"
)

func TestAccBlockStorageV1Volume_basic(t *testing.T) {
	var volume volumes.Volume

	var testAccBlockStorageV1Volume_bootable = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "volume_1" {
			region = "%s"
			name = "tf-test-volume-bootable"
			size = 5
			image_id = "%s"
		}`,
		os.Getenv("OS_REGION_NAME"), os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV1VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v1.volume_1", "name", "tf-test-volume"),
					testAccCheckBlockStorageV1VolumeMetadata(&volume, "foo", "bar"),
				),
			},
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v1.volume_1", "name", "tf-test-volume-updated"),
					testAccCheckBlockStorageV1VolumeMetadata(&volume, "foo", "bar"),
				),
			},
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_bootable,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v1.volume_1", "name", "tf-test-volume-bootable"),
				),
			},
		},
	})
}

func testAccCheckBlockStorageV1VolumeDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	blockStorageClient, err := config.blockStorageV1Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_blockstorage_volume_v1" {
			continue
		}

		_, err := volumes.Get(blockStorageClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Volume still exists")
		}
	}

	return nil
}

func testAccCheckBlockStorageV1VolumeExists(t *testing.T, n string, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		blockStorageClient, err := config.blockStorageV1Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
		}

		found, err := volumes.Get(blockStorageClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Volume not found")
		}

		*volume = *found

		return nil
	}
}

func testAccCheckBlockStorageV1VolumeDoesNotExist(t *testing.T, n string, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		blockStorageClient, err := config.blockStorageV1Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
		}

		_, err = volumes.Get(blockStorageClient, volume.ID).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return err
			}
			if errCode.Actual == 404 {
				return nil
			}
			return err
		}

		return fmt.Errorf("Volume still exists")
	}
}

func testAccCheckBlockStorageV1VolumeMetadata(
	volume *volumes.Volume, k string, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if volume.Metadata == nil {
			return fmt.Errorf("No metadata")
		}

		for key, value := range volume.Metadata {
			if k != key {
				continue
			}

			if v == value {
				return nil
			}

			return fmt.Errorf("Bad value for %s: %s", k, value)
		}

		return fmt.Errorf("Metadata not found: %s", k)
	}
}

var testAccBlockStorageV1Volume_basic = fmt.Sprintf(`
	resource "openstack_blockstorage_volume_v1" "volume_1" {
		region = "%s"
		name = "tf-test-volume"
		description = "first test volume"
		metadata{
			foo = "bar"
		}
		size = 1
	}`,
	OS_REGION_NAME)

var testAccBlockStorageV1Volume_update = fmt.Sprintf(`
	resource "openstack_blockstorage_volume_v1" "volume_1" {
		region = "%s"
		name = "tf-test-volume-updated"
		description = "first test volume"
		metadata{
			foo = "bar"
		}
		size = 1
	}`,
	OS_REGION_NAME)
