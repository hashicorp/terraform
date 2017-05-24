package rackspace

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	rsVolumes "github.com/rackspace/gophercloud/rackspace/blockstorage/v1/volumes"
)

func TestAccBlockStorageVolume_basic(t *testing.T) {
	var volume rsVolumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageVolume_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageVolumeExists(t, "rackspace_blockstorage_volume.volume_1", &volume),
					resource.TestCheckResourceAttr("rackspace_blockstorage_volume.volume_1", "name", "tf-test-volume"),
					testAccCheckBlockStorageVolumeMetadata(&volume, "foo", "bar"),
				),
			},
			resource.TestStep{
				Config: testAccBlockStorageVolume_update,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rackspace_blockstorage_volume.volume_1", "name", "tf-test-volume-updated"),
					testAccCheckBlockStorageVolumeMetadata(&volume, "foo", "bar"),
				),
			},
		},
	})
}

func testAccCheckBlockStorageVolumeDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	blockStorageClient, err := config.blockStorageClient(RS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating Rackspace block storage client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rackspace_blockstorage_volume" {
			continue
		}

		_, err := rsVolumes.Get(blockStorageClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Volume still exists")
		}
	}

	return nil
}

func testAccCheckBlockStorageVolumeExists(t *testing.T, n string, volume *rsVolumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		blockStorageClient, err := config.blockStorageClient(RS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating Rackspace block storage client: %s", err)
		}

		found, err := rsVolumes.Get(blockStorageClient, rs.Primary.ID).Extract()
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

func testAccCheckBlockStorageVolumeMetadata(
	volume *rsVolumes.Volume, k string, v string) resource.TestCheckFunc {
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

var testAccBlockStorageVolume_basic = fmt.Sprintf(`
                  resource "openstack_blockstorage_volume" "volume_1" {
                    region = "%s"
                    name = "tf-test-volume"
                    description = "first test volume"
                    metadata{
                      foo = "bar"
                    }
                    size = 1
                    }`,
	RS_REGION_NAME)

var testAccBlockStorageVolume_update = fmt.Sprintf(`
                      resource "openstack_blockstorage_volume" "volume_1" {
                        region = "%s"
                        name = "tf-test-volume-updated"
                        description = "first test volume"
                        metadata{
                          foo = "bar"
                        }
                        size = 1
                        }`,
	RS_REGION_NAME)
