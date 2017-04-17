package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
)

func TestAccBlockStorageVolumeAttachV2_basic(t *testing.T) {
	var va volumes.Attachment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageVolumeAttachV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageVolumeAttachV2_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageVolumeAttachV2Exists("openstack_blockstorage_volume_attach_v2.va_1", &va),
				),
			},
		},
	})
}

func TestAccBlockStorageVolumeAttachV2_timeout(t *testing.T) {
	var va volumes.Attachment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageVolumeAttachV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageVolumeAttachV2_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageVolumeAttachV2Exists("openstack_blockstorage_volume_attach_v2.va_1", &va),
				),
			},
		},
	})
}

func testAccCheckBlockStorageVolumeAttachV2Destroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	client, err := config.blockStorageV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_blockstorage_volume_attach_v2" {
			continue
		}

		volumeId, attachmentId, err := blockStorageVolumeAttachV2ParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		volume, err := volumes.Get(client, volumeId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return nil
			}
			return err
		}

		for _, v := range volume.Attachments {
			if attachmentId == v.AttachmentID {
				return fmt.Errorf("Volume attachment still exists")
			}
		}
	}

	return nil
}

func testAccCheckBlockStorageVolumeAttachV2Exists(n string, va *volumes.Attachment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		client, err := config.blockStorageV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
		}

		volumeId, attachmentId, err := blockStorageVolumeAttachV2ParseId(rs.Primary.ID)
		if err != nil {
			return err
		}

		volume, err := volumes.Get(client, volumeId).Extract()
		if err != nil {
			return err
		}

		var found bool
		for _, v := range volume.Attachments {
			if attachmentId == v.AttachmentID {
				found = true
				*va = v
			}
		}

		if !found {
			return fmt.Errorf("Volume Attachment not found")
		}

		return nil
	}
}

const testAccBlockStorageVolumeAttachV2_basic = `
resource "openstack_blockstorage_volume_v2" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_blockstorage_volume_attach_v2" "va_1" {
  volume_id = "${openstack_blockstorage_volume_v2.volume_1.id}"
  device = "auto"

  host_name = "devstack"
  ip_address = "192.168.255.10"
  initiator = "iqn.1993-08.org.debian:01:e9861fb1859"
  os_type = "linux2"
  platform = "x86_64"
}
`

const testAccBlockStorageVolumeAttachV2_timeout = `
resource "openstack_blockstorage_volume_v2" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_blockstorage_volume_attach_v2" "va_1" {
  volume_id = "${openstack_blockstorage_volume_v2.volume_1.id}"
  device = "auto"

  host_name = "devstack"
  ip_address = "192.168.255.10"
  initiator = "iqn.1993-08.org.debian:01:e9861fb1859"
  os_type = "linux2"
  platform = "x86_64"

  timeouts {
    create = "5m"
  }
}
`
