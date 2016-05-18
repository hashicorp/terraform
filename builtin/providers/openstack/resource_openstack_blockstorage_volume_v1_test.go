package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/blockstorage/v1/volumes"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
)

func TestAccBlockStorageV1Volume_basic(t *testing.T) {
	var volume volumes.Volume
	var testAccBlockStorageV1Volume_basic = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "volume_1" {
			name = "volume_1"
			description = "first test volume"
			metadata{
				foo = "bar"
			}
			size = 1
		}`)

	var testAccBlockStorageV1Volume_update = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "volume_1" {
			name = "volume_1-updated"
			description = "first test volume"
			metadata{
				foo = "bar"
			}
			size = 1
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV1VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v1.volume_1", "name", "volume_1"),
					testAccCheckBlockStorageV1VolumeMetadata(&volume, "foo", "bar"),
				),
			},
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v1.volume_1", "name", "volume_1-updated"),
					testAccCheckBlockStorageV1VolumeMetadata(&volume, "foo", "bar"),
				),
			},
		},
	})
}

func TestAccBlockStorageV1Volume_bootable(t *testing.T) {
	var volume volumes.Volume
	var testAccBlockStorageV1Volume_bootable = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "volume_1" {
			name = "volume_1-bootable"
			size = 5
			image_id = "%s"
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV1VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_bootable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					resource.TestCheckResourceAttr("openstack_blockstorage_volume_v1.volume_1", "name", "volume_1-bootable"),
				),
			},
		},
	})
}

func TestAccBlockStorageV1Volume_volumeAttach(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	var testAccBlockStorageV1Volume_volumeAttach = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]
		}

		resource "openstack_blockstorage_volume_v1" "volume_1" {
			name = "volume_1"
			size = 1
			attachment {
				instance_id = "${openstack_compute_instance_v2.instance_1.id}"
			}
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV1VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_volumeAttach,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckBlockStorageV1VolumeAttachment(&instance, &volume),
				),
			},
		},
	})
}

func TestAccBlockStorageV1Volume_volumeAttachPostCreation(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	var testAccBlockStorageV1Volume_volumeAttachPostCreationInstance = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]
		}`)

	var testAccBlockStorageV1Volume_volumeAttachPostCreationInstanceAndVolume = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]
		}

		resource "openstack_blockstorage_volume_v1" "volume_1" {
			name = "volume_1"
			size = 1

			attachment {
				instance_id = "${openstack_compute_instance_v2.instance_1.id}"
			}
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV1VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_volumeAttachPostCreationInstance,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
				),
			},
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_volumeAttachPostCreationInstanceAndVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckBlockStorageV1VolumeAttachment(&instance, &volume),
				),
			},
		},
	})
}

func TestAccBlockStorageV1Volume_volumeDetachPostCreation(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	var testAccBlockStorageV1Volume_volumeDetachPostCreation_1 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]
		}

		resource "openstack_blockstorage_volume_v1" "volume_1" {
			name = "volume_1"
			size = 1

			attachment {
				instance_id = "${openstack_compute_instance_v2.instance_1.id}"
			}
		}`)

	var testAccBlockStorageV1Volume_volumeDetachPostCreation_2 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]
		}

		resource "openstack_blockstorage_volume_v1" "volume_1" {
			depends_on = ["openstack_compute_instance_v2.instance_1"]
			name = "volume_1"
			size = 1
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageV1VolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_volumeDetachPostCreation_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckBlockStorageV1VolumeAttachment(&instance, &volume),
				),
			},
			resource.TestStep{
				Config: testAccBlockStorageV1Volume_volumeDetachPostCreation_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckBlockStorageV1VolumeNotAttached(&instance, &volume),
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

func testAccCheckBlockStorageV1VolumeAttachment(
	instance *servers.Server, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return err
		}

		allPages, err := volumeattach.List(computeClient, instance.ID).AllPages()
		if err != nil {
			return err
		}

		attachments, err := volumeattach.ExtractVolumeAttachments(allPages)
		if err != nil {
			return err
		}

		for _, attachment := range attachments {
			if attachment.VolumeID == volume.ID {
				return nil
			}
		}

		return fmt.Errorf("Volume not found: %s", volume.ID)
	}
}

func testAccCheckBlockStorageV1VolumeNotAttached(
	instance *servers.Server, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return err
		}

		allPages, err := volumeattach.List(computeClient, instance.ID).AllPages()
		if err != nil {
			return err
		}

		attachments, err := volumeattach.ExtractVolumeAttachments(allPages)
		if err != nil {
			return err
		}

		for _, attachment := range attachments {
			if attachment.VolumeID == volume.ID {
				return fmt.Errorf("Volume still attached: %s", volume.ID)
			}
		}

		return nil

	}
}
