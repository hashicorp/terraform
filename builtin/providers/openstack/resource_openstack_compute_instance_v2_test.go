package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v1/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"
)

func TestAccComputeV2Instance_basic(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceMetadata(&instance, "foo", "bar"),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "all_metadata.foo", "bar"),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "availability_zone", "nova"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttach(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttach,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists("openstack_blockstorage_volume_v1.vol_1", &volume),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttachPostCreation(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachPostCreation_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachPostCreation_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists("openstack_blockstorage_volume_v1.vol_1", &volume),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeDetachPostCreation(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachPostCreation_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists("openstack_blockstorage_volume_v1.vol_1", &volume),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachPostCreation_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists("openstack_blockstorage_volume_v1.vol_1", &volume),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumesDetached(&instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeDetachAdditionalVolumePostCreation(t *testing.T) {
	var instance servers.Server
	var volume_1 volumes.Volume
	var volume_2 volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachAdditionalVolumePostCreation_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_1),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_2),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachAdditionalVolumePostCreation_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumeDetached(
						&instance, "openstack_blockstorage_volume_v1.additional_volume"),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttachInstanceDelete(t *testing.T) {
	var instance servers.Server
	var volume_1 volumes.Volume
	var volume_2 volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachInstanceDelete_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_1),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_2),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachInstanceDelete_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceDoesNotExist(
						"openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceVolumeDetached(
						&instance, "openstack_blockstorage_volume_v1.root_volume"),
					testAccCheckComputeV2InstanceVolumeDetached(
						&instance, "openstack_blockstorage_volume_v1.additional_volume"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttachToNewInstance(t *testing.T) {
	var instance_1 servers.Server
	var instance_2 servers.Server
	var volume_1 volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachToNewInstance_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.volume_1", &volume_1),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_2", &instance_2),
					testAccCheckComputeV2InstanceVolumeDetached(
						&instance_2, "openstack_blockstorage_volume_v1.volume_1"),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance_1, &volume_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachToNewInstance_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.volume_1", &volume_1),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_2", &instance_2),
					testAccCheckComputeV2InstanceVolumeDetached(
						&instance_1, "openstack_blockstorage_volume_v1.volume_1"),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance_2, &volume_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_floatingIPAttachGlobally(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachGlobally,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists("openstack_compute_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_floatingIPAttachToNetwork(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachToNetwork,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists("openstack_compute_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_floatingIPAttachToNetworkAndChange(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachToNetworkAndChange_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists("openstack_compute_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachToNetworkAndChange_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists("openstack_compute_floatingip_v2.fip_2", &fip),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_secgroupMulti(t *testing.T) {
	var instance_1 servers.Server
	var secgroup_1 secgroups.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_secgroupMulti,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(
						"openstack_compute_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_secgroupMultiUpdate(t *testing.T) {
	var instance_1 servers.Server
	var secgroup_1, secgroup_2 secgroups.SecurityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_secgroupMultiUpdate_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(
						"openstack_compute_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckComputeV2SecGroupExists(
						"openstack_compute_secgroup_v2.secgroup_2", &secgroup_2),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_secgroupMultiUpdate_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(
						"openstack_compute_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckComputeV2SecGroupExists(
						"openstack_compute_secgroup_v2.secgroup_2", &secgroup_2),
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeImage(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeImage,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceBootVolumeAttachment(&instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeImageWithAttachedVolume(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeImageWithAttachedVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeVolume(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceBootVolumeAttachment(&instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeForceNew(t *testing.T) {
	var instance1_1 servers.Server
	var instance1_2 servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeForceNew_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance1_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeForceNew_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance1_2),
					testAccCheckComputeV2InstanceInstanceIDsDoNotMatch(&instance1_1, &instance1_2),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_blockDeviceNewVolume(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_blockDeviceNewVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_blockDeviceExistingVolume(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_blockDeviceExistingVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckBlockStorageV1VolumeExists(
						"openstack_blockstorage_volume_v1.volume_1", &volume),
				),
			},
		},
	})
}

// TODO: verify the personality really exists on the instance.
func TestAccComputeV2Instance_personality(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_personality,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_multiEphemeral(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_multiEphemeral,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_accessIPv4(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_accessIPv4,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "access_ip_v4", "192.168.1.100"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_changeFixedIP(t *testing.T) {
	var instance1_1 servers.Server
	var instance1_2 servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_changeFixedIP_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance1_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_changeFixedIP_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(
						"openstack_compute_instance_v2.instance_1", &instance1_2),
					testAccCheckComputeV2InstanceInstanceIDsDoNotMatch(&instance1_1, &instance1_2),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_stopBeforeDestroy(t *testing.T) {
	var instance servers.Server
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_stopBeforeDestroy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_metadataRemove(t *testing.T) {
	var instance servers.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_metadataRemove_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeV2InstanceMetadata(&instance, "abc", "def"),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "all_metadata.foo", "bar"),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "all_metadata.abc", "def"),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_metadataRemove_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceMetadata(&instance, "foo", "bar"),
					testAccCheckComputeV2InstanceMetadata(&instance, "ghi", "jkl"),
					testAccCheckComputeV2InstanceNoMetadataKey(&instance, "abc"),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "all_metadata.foo", "bar"),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "all_metadata.ghi", "jkl"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_forceDelete(t *testing.T) {
	var instance servers.Server
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_forceDelete,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_timeout(t *testing.T) {
	var instance servers.Server
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_timeout,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_networkNameToID(t *testing.T) {
	var instance servers.Server
	var network networks.Network
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_networkNameToID,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckNetworkingV2NetworkExists("openstack_networking_network_v2.network_1", &network),
					resource.TestCheckResourceAttrPtr(
						"openstack_compute_instance_v2.instance_1", "network.1.uuid", &network.ID),
				),
			},
		},
	})
}

func testAccCheckComputeV2InstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_compute_instance_v2" {
			continue
		}

		server, err := servers.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			if server.Status != "SOFT_DELETED" {
				return fmt.Errorf("Instance still exists")
			}
		}
	}

	return nil
}

func testAccCheckComputeV2InstanceExists(n string, instance *servers.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenStack compute client: %s", err)
		}

		found, err := servers.Get(computeClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}

		*instance = *found

		return nil
	}
}

func testAccCheckComputeV2InstanceDoesNotExist(n string, instance *servers.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating OpenStack compute client: %s", err)
		}

		_, err = servers.Get(computeClient, instance.ID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return nil
			}
			return err
		}

		return fmt.Errorf("Instance still exists")
	}
}

func testAccCheckComputeV2InstanceMetadata(
	instance *servers.Server, k string, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Metadata == nil {
			return fmt.Errorf("No metadata")
		}

		for key, value := range instance.Metadata {
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

func testAccCheckComputeV2InstanceNoMetadataKey(
	instance *servers.Server, k string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Metadata == nil {
			return nil
		}

		for key, _ := range instance.Metadata {
			if k == key {
				return fmt.Errorf("Metadata found: %s", k)
			}
		}

		return nil
	}
}

func testAccCheckComputeV2InstanceVolumeAttachment(
	instance *servers.Server, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var attachments []volumeattach.VolumeAttachment

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return err
		}

		err = volumeattach.List(computeClient, instance.ID).EachPage(
			func(page pagination.Page) (bool, error) {

				actual, err := volumeattach.ExtractVolumeAttachments(page)
				if err != nil {
					return false, fmt.Errorf("Unable to lookup attachment: %s", err)
				}

				attachments = actual
				return true, nil
			})

		for _, attachment := range attachments {
			if attachment.VolumeID == volume.ID {
				return nil
			}
		}

		return fmt.Errorf("Volume not found: %s", volume.ID)
	}
}

func testAccCheckComputeV2InstanceVolumesDetached(instance *servers.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var attachments []volumeattach.VolumeAttachment

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return err
		}

		err = volumeattach.List(computeClient, instance.ID).EachPage(
			func(page pagination.Page) (bool, error) {

				actual, err := volumeattach.ExtractVolumeAttachments(page)
				if err != nil {
					return false, fmt.Errorf("Unable to lookup attachment: %s", err)
				}

				attachments = actual
				return true, nil
			})

		if len(attachments) > 0 {
			return fmt.Errorf("Volumes are still attached.")
		}

		return nil
	}
}

func testAccCheckComputeV2InstanceBootVolumeAttachment(
	instance *servers.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var attachments []volumeattach.VolumeAttachment

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return err
		}

		err = volumeattach.List(computeClient, instance.ID).EachPage(
			func(page pagination.Page) (bool, error) {

				actual, err := volumeattach.ExtractVolumeAttachments(page)
				if err != nil {
					return false, fmt.Errorf("Unable to lookup attachment: %s", err)
				}

				attachments = actual
				return true, nil
			})

		if len(attachments) == 1 {
			return nil
		}

		return fmt.Errorf("No attached volume found.")
	}
}

func testAccCheckComputeV2InstanceFloatingIPAttach(
	instance *servers.Server, fip *floatingips.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if fip.InstanceID == instance.ID {
			return nil
		}

		return fmt.Errorf("Floating IP %s was not attached to instance %s", fip.ID, instance.ID)
	}
}

func testAccCheckComputeV2InstanceInstanceIDsDoNotMatch(
	instance1, instance2 *servers.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance1.ID == instance2.ID {
			return fmt.Errorf("Instance was not recreated.")
		}

		return nil
	}
}

func testAccCheckComputeV2InstanceVolumeDetached(instance *servers.Server, volume_id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var attachments []volumeattach.VolumeAttachment

		rs, ok := s.RootModule().Resources[volume_id]
		if !ok {
			return fmt.Errorf("Not found: %s", volume_id)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return err
		}

		err = volumeattach.List(computeClient, instance.ID).EachPage(
			func(page pagination.Page) (bool, error) {
				actual, err := volumeattach.ExtractVolumeAttachments(page)
				if err != nil {
					return false, fmt.Errorf("Unable to lookup attachment: %s", err)
				}

				attachments = actual
				return true, nil
			})

		for _, attachment := range attachments {
			if attachment.VolumeID == rs.Primary.ID {
				return fmt.Errorf("Volume is still attached.")
			}
		}

		return nil
	}
}

const testAccComputeV2Instance_basic = `
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  metadata {
    foo = "bar"
  }
}
`

const testAccComputeV2Instance_volumeAttach = `
resource "openstack_blockstorage_volume_v1" "vol_1" {
  name = "vol_1"
 	size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  volume {
    volume_id = "${openstack_blockstorage_volume_v1.vol_1.id}"
  }
}
`

const testAccComputeV2Instance_volumeAttachPostCreation_1 = `
resource "openstack_compute_instance_v2" "instance_1" {
	name = "instance_1"
	security_groups = ["default"]
}
`

const testAccComputeV2Instance_volumeAttachPostCreation_2 = `
resource "openstack_blockstorage_volume_v1" "vol_1" {
  name = "vol_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  volume {
   volume_id = "${openstack_blockstorage_volume_v1.vol_1.id}"
  }
}
`

const testAccComputeV2Instance_volumeDetachPostCreation_1 = `
resource "openstack_blockstorage_volume_v1" "vol_1" {
  name = "vol_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  volume {
    volume_id = "${openstack_blockstorage_volume_v1.vol_1.id}"
  }
}
`

const testAccComputeV2Instance_volumeDetachPostCreation_2 = `
resource "openstack_blockstorage_volume_v1" "vol_1" {
  name = "vol_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
}
`

var testAccComputeV2Instance_volumeDetachAdditionalVolumePostCreation_1 = fmt.Sprintf(`
resource "openstack_blockstorage_volume_v1" "root_volume" {
  name = "root_volume"
  size = 1
  image_id = "%s"
}

resource "openstack_blockstorage_volume_v1" "additional_volume" {
  name = "additional_volume"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  block_device {
    uuid = "${openstack_blockstorage_volume_v1.root_volume.id}"
    source_type = "volume"
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = false
  }

  volume {
    volume_id = "${openstack_blockstorage_volume_v1.additional_volume.id}"
	}
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_volumeDetachAdditionalVolumePostCreation_2 = fmt.Sprintf(`
resource "openstack_blockstorage_volume_v1" "root_volume" {
  name = "root_volume"
  size = 1
  image_id = "%s"
}

resource "openstack_blockstorage_volume_v1" "additional_volume" {
  name = "additional_volume"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  block_device {
    uuid = "${openstack_blockstorage_volume_v1.root_volume.id}"
    source_type = "volume"
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = false
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_volumeAttachInstanceDelete_1 = fmt.Sprintf(`
resource "openstack_blockstorage_volume_v1" "root_volume" {
  name = "root_volume"
  size = 1
  image_id = "%s"
}

resource "openstack_blockstorage_volume_v1" "additional_volume" {
  name = "additional_volume"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  block_device {
    uuid = "${openstack_blockstorage_volume_v1.root_volume.id}"
    source_type = "volume"
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = false
  }

  volume {
    volume_id = "${openstack_blockstorage_volume_v1.additional_volume.id}"
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_volumeAttachInstanceDelete_2 = fmt.Sprintf(`
resource "openstack_blockstorage_volume_v1" "root_volume" {
  name = "root_volume"
  size = 1
  image_id = "%s"
}

resource "openstack_blockstorage_volume_v1" "additional_volume" {
  name = "additional_volume"
  size = 1
}
`, OS_IMAGE_ID)

const testAccComputeV2Instance_volumeAttachToNewInstance_1 = `
resource "openstack_blockstorage_volume_v1" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  volume {
    volume_id = "${openstack_blockstorage_volume_v1.volume_1.id}"
  }
}

resource "openstack_compute_instance_v2" "instance_2" {
  depends_on = ["openstack_compute_instance_v2.instance_1"]
  name = "instance_2"
  security_groups = ["default"]
}
`

const testAccComputeV2Instance_volumeAttachToNewInstance_2 = `
resource "openstack_blockstorage_volume_v1" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
}

resource "openstack_compute_instance_v2" "instance_2" {
  depends_on = ["openstack_compute_instance_v2.instance_1"]
  name = "instance_2"
  security_groups = ["default"]

  volume {
    volume_id = "${openstack_blockstorage_volume_v1.volume_1.id}"
  }
}
	`

const testAccComputeV2Instance_floatingIPAttachGlobally = `
resource "openstack_compute_floatingip_v2" "fip_1" {
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  floating_ip = "${openstack_compute_floatingip_v2.fip_1.address}"
}
`

var testAccComputeV2Instance_floatingIPAttachToNetwork = fmt.Sprintf(`
resource "openstack_compute_floatingip_v2" "fip_1" {
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  network {
    uuid = "%s"
    floating_ip = "${openstack_compute_floatingip_v2.fip_1.address}"
    access_network = true
  }
}
`, OS_NETWORK_ID)

var testAccComputeV2Instance_floatingIPAttachToNetworkAndChange_1 = fmt.Sprintf(`
resource "openstack_compute_floatingip_v2" "fip_1" {
}

resource "openstack_compute_floatingip_v2" "fip_2" {
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  network {
    uuid = "%s"
    floating_ip = "${openstack_compute_floatingip_v2.fip_1.address}"
    access_network = true
  }
}
`, OS_NETWORK_ID)

var testAccComputeV2Instance_floatingIPAttachToNetworkAndChange_2 = fmt.Sprintf(`
resource "openstack_compute_floatingip_v2" "fip_1" {
}

resource "openstack_compute_floatingip_v2" "fip_2" {
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  network {
    uuid = "%s"
    floating_ip = "${openstack_compute_floatingip_v2.fip_2.address}"
    access_network = true
  }
}
`, OS_NETWORK_ID)

const testAccComputeV2Instance_secgroupMulti = `
resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "a security group"
  rule {
    from_port = 22
    to_port = 22
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}"]
}
`

const testAccComputeV2Instance_secgroupMultiUpdate_1 = `
resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "a security group"
  rule {
    from_port = 22
    to_port = 22
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_compute_secgroup_v2" "secgroup_2" {
  name = "secgroup_2"
  description = "another security group"
  rule {
    from_port = 80
    to_port = 80
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
}
`

const testAccComputeV2Instance_secgroupMultiUpdate_2 = `
resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "a security group"
  rule {
    from_port = 22
    to_port = 22
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_compute_secgroup_v2" "secgroup_2" {
  name = "secgroup_2"
  description = "another security group"
  rule {
    from_port = 80
    to_port = 80
    ip_protocol = "tcp"
    cidr = "0.0.0.0/0"
  }
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}", "${openstack_compute_secgroup_v2.secgroup_2.name}"]
}
`

var testAccComputeV2Instance_bootFromVolumeImage = fmt.Sprintf(`
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  block_device {
    uuid = "%s"
    source_type = "image"
    volume_size = 5
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = true
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_bootFromVolumeImageWithAttachedVolume = fmt.Sprintf(`
resource "openstack_blockstorage_volume_v1" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  block_device {
    uuid = "%s"
    source_type = "image"
    volume_size = 2
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = true
  }

  volume {
    volume_id = "${openstack_blockstorage_volume_v1.volume_1.id}"
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_bootFromVolumeVolume = fmt.Sprintf(`
resource "openstack_blockstorage_volume_v1" "vol_1" {
  name = "vol_1"
  size = 5
  image_id = "%s"
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  block_device {
    uuid = "${openstack_blockstorage_volume_v1.vol_1.id}"
    source_type = "volume"
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = true
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_bootFromVolumeForceNew_1 = fmt.Sprintf(`
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  block_device {
    uuid = "%s"
    source_type = "image"
    volume_size = 5
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = true
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_bootFromVolumeForceNew_2 = fmt.Sprintf(`
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  block_device {
    uuid = "%s"
    source_type = "image"
    volume_size = 4
    boot_index = 0
    destination_type = "volume"
    delete_on_termination = true
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_blockDeviceNewVolume = fmt.Sprintf(`
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  block_device {
    uuid = "%s"
    source_type = "image"
    destination_type = "local"
    boot_index = 0
    delete_on_termination = true
  }
  block_device {
    source_type = "blank"
    destination_type = "volume"
    volume_size = 1
    boot_index = 1
    delete_on_termination = true
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_blockDeviceExistingVolume = fmt.Sprintf(`
resource "openstack_blockstorage_volume_v1" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  block_device {
    uuid = "%s"
    source_type = "image"
    destination_type = "local"
    boot_index = 0
    delete_on_termination = true
  }
  block_device {
    uuid = "${openstack_blockstorage_volume_v1.volume_1.id}"
    source_type = "volume"
    destination_type = "volume"
    boot_index = 1
    delete_on_termination = true
  }
}
`, OS_IMAGE_ID)

const testAccComputeV2Instance_personality = `
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  personality {
    file = "/tmp/foobar.txt"
    content = "happy"
  }
  personality {
    file = "/tmp/barfoo.txt"
    content = "angry"
  }
}
`

var testAccComputeV2Instance_multiEphemeral = fmt.Sprintf(`
resource "openstack_compute_instance_v2" "instance_1" {
  name = "terraform-test"
  security_groups = ["default"]
  block_device {
    boot_index = 0
    delete_on_termination = true
    destination_type = "local"
    source_type = "image"
    uuid = "%s"
  }
  block_device {
    boot_index = -1
    delete_on_termination = true
    destination_type = "local"
    source_type = "blank"
    volume_size = 1
  }
  block_device {
    boot_index = -1
    delete_on_termination = true
    destination_type = "local"
    source_type = "blank"
    volume_size = 1
  }
}
`, OS_IMAGE_ID)

var testAccComputeV2Instance_accessIPv4 = fmt.Sprintf(`
resource "openstack_compute_floatingip_v2" "myip" {
}

resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  network_id = "${openstack_networking_network_v2.network_1.id}"
  cidr = "192.168.1.0/24"
  ip_version = 4
  enable_dhcp = true
  no_gateway = true
}

resource "openstack_compute_instance_v2" "instance_1" {
  depends_on = ["openstack_networking_subnet_v2.subnet_1"]

  name = "instance_1"
  security_groups = ["default"]
  floating_ip = "${openstack_compute_floatingip_v2.myip.address}"

  network {
    uuid = "%s"
  }

  network {
    uuid = "${openstack_networking_network_v2.network_1.id}"
    fixed_ip_v4 = "192.168.1.100"
    access_network = true
  }
}
`, OS_NETWORK_ID)

var testAccComputeV2Instance_changeFixedIP_1 = fmt.Sprintf(`
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  network {
    uuid = "%s"
    fixed_ip_v4 = "10.0.0.24"
  }
}
`, OS_NETWORK_ID)

var testAccComputeV2Instance_changeFixedIP_2 = fmt.Sprintf(`
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  network {
    uuid = "%s"
    fixed_ip_v4 = "10.0.0.25"
  }
}
`, OS_NETWORK_ID)

const testAccComputeV2Instance_stopBeforeDestroy = `
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  stop_before_destroy = true
}
`

const testAccComputeV2Instance_metadataRemove_1 = `
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  metadata {
    foo = "bar"
    abc = "def"
  }
}
`

const testAccComputeV2Instance_metadataRemove_2 = `
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  metadata {
    foo = "bar"
    ghi = "jkl"
  }
}
`

const testAccComputeV2Instance_forceDelete = `
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  force_delete = true
}
`

const testAccComputeV2Instance_timeout = `
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  timeouts {
    create = "10m"
  }
}
`

var testAccComputeV2Instance_networkNameToID = fmt.Sprintf(`
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  network_id = "${openstack_networking_network_v2.network_1.id}"
  cidr = "192.168.1.0/24"
  ip_version = 4
  enable_dhcp = true
  no_gateway = true
}

resource "openstack_compute_instance_v2" "instance_1" {
  depends_on = ["openstack_networking_subnet_v2.subnet_1"]

  name = "instance_1"
  security_groups = ["default"]

  network {
    uuid = "%s"
  }

  network {
    name = "${openstack_networking_network_v2.network_1.name}"
  }

}
`, OS_NETWORK_ID)
