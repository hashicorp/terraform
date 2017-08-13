package netapp

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCloudVolume_nfs_import(t *testing.T) {
	resourceName := "netapp_cloud_volume.vsa-nfs-volume"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_nfs_vsa,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCloudVolume_cifs_import(t *testing.T) {
	resourceName := "netapp_cloud_volume.vsa-cifs-volume"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudVolume_cifs_vsa,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
