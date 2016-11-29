package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
)

func TestAccBlockStorageV2Volume_basic(t *testing.T) {
	var volume volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV2VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV2Volume_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV2VolumeExists(t, "openstack_blockstorage_volume_v2.volume_1", &volume),
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v2.volume_1", "name", "volume_1"),
					testAccCheckBlockStorageV2VolumeMetadata(&volume, "foo", "bar"),
				),
			},
			resource.TestStep{
				Config: testAccBlockStorageV2Volume_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV2VolumeExists(t, "openstack_blockstorage_volume_v2.volume_1", &volume),
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v2.volume_1", "name", "volume_1-updated"),
					testAccCheckBlockStorageV2VolumeMetadata(&volume, "foo", "bar"),
				),
			},
		},
	})
}

func TestAccBlockStorageV2Volume_bootable(t *testing.T) {
	var volume volumes.Volume

	var testAccBlockStorageV2Volume_bootable = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v2" "volume_1" {
			name = "volume_1-bootable"
			size = 5
			image_id = "%s"
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV2VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV2Volume_bootable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV2VolumeExists(t, "openstack_blockstorage_volume_v2.volume_1", &volume),
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v2.volume_1", "name", "volume_1-bootable"),
				),
			},
		},
	})
}

func testAccCheckBlockStorageV2VolumeDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	blockStorageClient, err := config.blockStorageV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_blockstorage_volume_v2" {
			continue
		}

		_, err := volumes.Get(blockStorageClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Volume still exists")
		}
	}

	return nil
}

func testAccCheckBlockStorageV2VolumeExists(t *testing.T, n string, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		blockStorageClient, err := config.blockStorageV2Client(OS_REGION_NAME)
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

func testAccCheckBlockStorageV2VolumeDoesNotExist(t *testing.T, n string, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		blockStorageClient, err := config.blockStorageV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
		}

		_, err = volumes.Get(blockStorageClient, volume.ID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return nil
			}
			return err
		}

		return fmt.Errorf("Volume still exists")
	}
}

func testAccCheckBlockStorageV2VolumeMetadata(
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

var testAccBlockStorageV2Volume_basic = fmt.Sprintf(`
	resource "openstack_blockstorage_volume_v2" "volume_1" {
		name = "volume_1"
		description = "first test volume"
		metadata {
			foo = "bar"
		}
		size = 1
	}`)

var testAccBlockStorageV2Volume_update = fmt.Sprintf(`
	resource "openstack_blockstorage_volume_v2" "volume_1" {
		name = "volume_1-updated"
		description = "first test volume"
		metadata {
			foo = "bar"
		}
		size = 1
	}`)
