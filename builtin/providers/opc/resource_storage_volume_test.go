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

func TestAccOPCStorageVolume_ImageListEntry(t *testing.T) {
	volumeResourceName := "opc_compute_storage_volume.test"
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeImageListEntry, ri, ri)

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
  description = "Provider Acceptance Tests Storage Volume Bootable"
}

resource "opc_compute_image_list_entry" "test" {
  name           = "${opc_compute_image_list.test.name}"
  machine_images = [ "/oracle/public/oel_6.7_apaas_16.4.5_1610211300" ]
  version        = 1
}

resource "opc_compute_storage_volume" "test" {
  name             = "test-acc-stor-vol-bootable-%d"
  description      = "Provider Acceptance Tests Storage Volume Bootable"
  size             = 20
  tags             = ["bar", "foo"]
  bootable         = true
  image_list       = "${opc_compute_image_list.test.name}"
  image_list_entry = "${opc_compute_image_list_entry.test.version}"
}
`

const testAccStorageVolumeImageListEntry = `
resource "opc_compute_image_list" "test" {
  name        = "test-acc-stor-vol-bootable-image-list-%d"
  description = "Provider Acceptance Tests Storage Volume Image List Entry"
}

resource "opc_compute_image_list_entry" "test" {
  name           = "${opc_compute_image_list.test.name}"
  machine_images = [ "/oracle/public/oel_6.7_apaas_16.4.5_1610211300" ]
  version        = 1
}

resource "opc_compute_storage_volume" "test" {
  name             = "test-acc-stor-vol-bootable-%d"
  description      = "Provider Acceptance Tests Storage Volume Image List Entry"
  size             = 20
  tags             = ["bar", "foo"]
  image_list_entry = "${opc_compute_image_list_entry.test.version}"
}
`

const testAccStorageVolumeBasicMaxSize = `
resource "opc_compute_storage_volume" "test" {
  name = "test-acc-stor-vol-%d"
  description = "Provider Acceptance Tests Storage Volume Max Size"
  size = 2048
}
`
