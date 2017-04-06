package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCStorageVolume_Basic(t *testing.T) {
	volumeResourceName := "opc_compute_storage_volume.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeBasic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeExists),
				),
			},
		},
	})
}

func TestAccOPCStorageVolume_Complete(t *testing.T) {
	volumeResourceName := "opc_compute_storage_volume.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeComplete, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeExists),
				),
			},
		},
	})
}

func TestAccOPCStorageVolume_MaxSize(t *testing.T) {
	volumeResourceName := "opc_compute_storage_volume.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeBasicMaxSize, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeExists),
				),
			},
		},
	})
}

func TestAccOPCStorageVolume_Update(t *testing.T) {
	volumeResourceName := "opc_compute_storage_volume.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeComplete, ri)
	updatedConfig := fmt.Sprintf(testAccStorageVolumeUpdated, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeExists),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeExists),
				),
			},
		},
	})
}

func TestAccOPCStorageVolume_Bootable(t *testing.T) {
	volumeResourceName := "opc_compute_storage_volume.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeBootable, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeExists),
				),
			},
		},
	})
}

func TestAccOPCStorageVolume_FromSnapshot(t *testing.T) {
	volumeResourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolumeFromSnapshot(rInt),
				Check: resource.ComposeTestCheckFunc(
					opcResourceCheck(volumeResourceName, testAccCheckStorageVolumeExists),
					resource.TestCheckResourceAttr(volumeResourceName, "name", fmt.Sprintf("test-acc-stor-vol-final-%d", rInt)),
					resource.TestCheckResourceAttrSet(volumeResourceName, "snapshot"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "snapshot_id"),
					resource.TestCheckResourceAttr(volumeResourceName, "size", "5"),
				),
			},
		},
	})
}

func testAccCheckStorageVolumeExists(state *OPCResourceState) error {
	sv := state.Client.StorageVolumes()
	volumeName := state.Attributes["name"]

	input := compute.GetStorageVolumeInput{
		Name: volumeName,
	}
	info, err := sv.GetStorageVolume(&input)
	if err != nil {
		return fmt.Errorf("Error retrieving state of volume %s: %s", volumeName, err)
	}

	if info == nil {
		return fmt.Errorf("No info found for volume %s", volumeName)
	}

	return nil
}

func testAccCheckStorageVolumeDestroyed(state *OPCResourceState) error {
	sv := state.Client.StorageVolumes()

	volumeName := state.Attributes["name"]

	input := compute.GetStorageVolumeInput{
		Name: volumeName,
	}
	info, err := sv.GetStorageVolume(&input)
	if err != nil {
		return fmt.Errorf("Error retrieving state of volume %s: %s", volumeName, err)
	}

	if info != nil {
		return fmt.Errorf("Volume %s still exists", volumeName)
	}

	return nil
}

const testAccStorageVolumeBasic = `
resource "opc_compute_storage_volume" "test" {
  name = "test-acc-stor-vol-%d"
  size = 1
}
`

const testAccStorageVolumeComplete = `
resource "opc_compute_storage_volume" "test" {
  name        = "test-acc-stor-vol-%d"
  description = "Provider Acceptance Tests Storage Volume Initial"
  size        = 2
  tags        = ["foo"]
}
`

const testAccStorageVolumeUpdated = `
resource "opc_compute_storage_volume" "test" {
  name        = "test-acc-stor-vol-%d"
  description = "Provider Acceptance Tests Storage Volume Updated"
  size        = 2
  tags        = ["bar", "foo"]
}
`

const testAccStorageVolumeBootable = `
resource "opc_compute_image_list" "test" {
  name        = "test-acc-stor-vol-bootable-image-list-%d"
  description = "Provider Acceptance Tests Storage Volume"
}

resource "opc_compute_storage_volume" "test" {
  name        = "test-acc-stor-vol-bootable-%d"
  description = "Provider Acceptance Tests Storage Volume"
  size        = 2
  tags        = ["bar", "foo"]
  bootable {
  	image_list = "${opc_compute_image_list.test.name}"
  }
}
`

const testAccStorageVolumeBasicMaxSize = `
resource "opc_compute_storage_volume" "test" {
  name = "test-acc-stor-vol-%d"
  description = "Provider Acceptance Tests Storage Volume Max Size"
  size = 2048
}
`

func testAccStorageVolumeFromSnapshot(rInt int) string {
	return fmt.Sprintf(`
// Initial Storage Volume to create snapshot with
resource "opc_compute_storage_volume" "foo" {
  name = "test-acc-stor-vol-%d"
  description = "Acc Test intermediary storage volume for snapshot"
  size = 5
}

resource "opc_compute_storage_volume_snapshot" "foo" {
		description = "testing-acc"
		name = "test-acc-stor-snapshot-%d"
		collocated = true
		volume = "${opc_compute_storage_volume.foo.name}"
}

// Create storage volume from snapshot
resource "opc_compute_storage_volume" "test" {
  name = "test-acc-stor-vol-final-%d"
  description = "storage volume from snapshot"
  size = 5
  snapshot_id = "${opc_compute_storage_volume_snapshot.foo.snapshot_id}"
}
`, rInt, rInt, rInt)
}
