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
