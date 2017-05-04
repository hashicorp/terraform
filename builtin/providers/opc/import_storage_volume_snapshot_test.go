package opc

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCStorageVolumeSnapshot_importBasic(t *testing.T) {
	resourceName := "opc_compute_storage_volume_snapshot.test"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeSnapshotDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolumeSnapshot_basic(rInt),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
