package opc

import (
	"testing"

	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCStorageVolume_importBasic(t *testing.T) {
	resourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeBasic, rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOPCStorageVolume_importComplete(t *testing.T) {
	resourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeComplete, rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOPCStorageVolume_importMaxSize(t *testing.T) {
	resourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeBasicMaxSize, rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOPCStorageVolume_importBootable(t *testing.T) {
	resourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeBootable, rInt, rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOPCStorageVolume_importImageListEntry(t *testing.T) {
	resourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()
	config := fmt.Sprintf(testAccStorageVolumeImageListEntry, rInt, rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOPCStorageVolume_importLowLatency(t *testing.T) {
	resourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()
	config := testAccStorageVolumeLowLatency(rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOPCStorageVolume_importFromSnapshot(t *testing.T) {
	resourceName := "opc_compute_storage_volume.test"
	rInt := acctest.RandInt()
	config := testAccStorageVolumeFromSnapshot(rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: opcResourceCheck(resourceName, testAccCheckStorageVolumeDestroyed),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
