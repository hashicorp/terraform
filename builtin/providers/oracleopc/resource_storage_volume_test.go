package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccOPCStorageVolume_Basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: opcResourceCheck(
			"opc_compute_storage_volume.test_volume",
			testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccStorageVolumeBasic,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(
						"opc_compute_storage_volume.test_volume",
						testAccCheckStorageVolumeExists),
				),
			},
		},
	})
}

func testAccCheckStorageVolumeExists(state *OPCResourceState) error {
	sv := state.StorageVolumes()
	volumeName := state.Attributes["name"]

	info, err := sv.GetStorageVolume(volumeName)
	if err != nil {
		return fmt.Errorf("Error retrieving state of volume %s: %s", volumeName, err)
	}

	if len(info.Result) == 0 {
		return fmt.Errorf("No info found for volume %s", volumeName)
	}

	return nil
}

func testAccCheckStorageVolumeDestroyed(state *OPCResourceState) error {
	sv := state.StorageVolumes()

	volumeName := state.Attributes["name"]

	info, err := sv.GetStorageVolume(volumeName)
	if err != nil {
		return fmt.Errorf("Error retrieving state of volume %s: %s", volumeName, err)
	}

	if len(info.Result) != 0 {
		return fmt.Errorf("Volume %s still exists", volumeName)
	}

	return nil
}

const testAccStorageVolumeBasic = `
resource "opc_compute_storage_volume" "test_volume" {
	size = "3g"
	description = "My volume"
	name = "test_volume_b"
	tags = ["foo", "bar", "baz"]
}
`
