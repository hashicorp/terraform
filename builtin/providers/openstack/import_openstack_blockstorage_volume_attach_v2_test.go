package openstack

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccBlockStorageVolumeAttachV2_importBasic(t *testing.T) {
	resourceName := "openstack_blockstorage_volume_attach_v2.va_1"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBlockStorageVolumeAttachV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccBlockStorageVolumeAttachV2_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
