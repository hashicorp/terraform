package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v1/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
)

func TestAccComputeV2Instance_basic(t *testing.T) {
	var instance servers.Server
	var testAccComputeV2Instance_basic = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			network {
				uuid = "%s"
			}
			metadata {
				foo = "bar"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceMetadata(&instance, "foo", "bar"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttach(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	var testAccComputeV2Instance_volumeAttach = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "myvol" {
			name = "myvol"
			size = 1
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			volume {
				volume_id = "${openstack_blockstorage_volume_v1.myvol.id}"
			}
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttach,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.myvol", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttachPostCreation(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	var testAccComputeV2Instance_volumeAttachPostCreationInstance = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
		}`)

	var testAccComputeV2Instance_volumeAttachPostCreationInstanceAndVolume = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "myvol" {
			name = "myvol"
			size = 1
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			volume {
				volume_id = "${openstack_blockstorage_volume_v1.myvol.id}"
			}
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachPostCreationInstance,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachPostCreationInstanceAndVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.myvol", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeDetachPostCreation(t *testing.T) {
	var instance servers.Server
	var volume volumes.Volume

	var testAccComputeV2Instance_volumeDetachPostCreationInstanceAndVolume = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "myvol" {
			name = "myvol"
			size = 1
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			volume {
				volume_id = "${openstack_blockstorage_volume_v1.myvol.id}"
			}
		}`)

	var testAccComputeV2Instance_volumeDetachPostCreationInstance = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "myvol" {
			name = "myvol"
			size = 1
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachPostCreationInstanceAndVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.myvol", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachPostCreationInstance,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.myvol", &volume),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
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

	var testAccComputeV2Instance_volumeDetachAdditionalVolumePostCreationInstanceAndVolume = fmt.Sprintf(`

		resource "openstack_blockstorage_volume_v1" "root_volume" {
			name = "root_volume"
			size = 1
			image_id = "%s"
		}

		resource "openstack_blockstorage_volume_v1" "additional_volume" {
			name = "additional_volume"
			size = 1
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
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
		}`,
		os.Getenv("OS_IMAGE_ID"))

	var testAccComputeV2Instance_volumeDetachPostCreationInstance = fmt.Sprintf(`

		resource "openstack_blockstorage_volume_v1" "root_volume" {
			name = "root_volume"
			size = 1
			image_id = "%s"
		}

		resource "openstack_blockstorage_volume_v1" "additional_volume" {
			name = "additional_volume"
			size = 1
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]

			block_device {
				uuid = "${openstack_blockstorage_volume_v1.root_volume.id}"
				source_type = "volume"
				boot_index = 0
				destination_type = "volume"
				delete_on_termination = false
			}
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachAdditionalVolumePostCreationInstanceAndVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_1),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_2),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeDetachPostCreationInstance,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_1),
					testAccCheckComputeV2InstanceVolumeDetached(&instance, "openstack_blockstorage_volume_v1.additional_volume"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttachInstanceDelete(t *testing.T) {
	var instance servers.Server
	var volume_1 volumes.Volume
	var volume_2 volumes.Volume

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

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
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
		}`,
		os.Getenv("OS_IMAGE_ID"))

	var testAccComputeV2Instance_volumeAttachInstanceDelete_2 = fmt.Sprintf(`
		resource "openstack_blockstorage_volume_v1" "root_volume" {
			name = "root_volume"
			size = 1
			image_id = "%s"
		}

		resource "openstack_blockstorage_volume_v1" "additional_volume" {
			name = "additional_volume"
			size = 1
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachInstanceDelete_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_1),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance, &volume_2),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachInstanceDelete_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.root_volume", &volume_1),
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.additional_volume", &volume_2),
					testAccCheckComputeV2InstanceDoesNotExist(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceVolumeDetached(&instance, "openstack_blockstorage_volume_v1.root_volume"),
					testAccCheckComputeV2InstanceVolumeDetached(&instance, "openstack_blockstorage_volume_v1.additional_volume"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_volumeAttachToNewInstance(t *testing.T) {
	var instance_1 servers.Server
	var instance_2 servers.Server
	var volume_1 volumes.Volume

	var testAccComputeV2Instance_volumeAttachToNewInstance_1 = fmt.Sprintf(`
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
		}`)

	var testAccComputeV2Instance_volumeAttachToNewInstance_2 = fmt.Sprintf(`
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
			}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachToNewInstance_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_2", &instance_2),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance_1, &volume_1),
					testAccCheckComputeV2InstanceVolumeDetached(&instance_2, "openstack_blockstorage_volume_v1.volume_1"),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_volumeAttachToNewInstance_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_2", &instance_2),
					testAccCheckComputeV2InstanceVolumeDetached(&instance_1, "openstack_blockstorage_volume_v1.volume_1"),
					testAccCheckComputeV2InstanceVolumeAttachment(&instance_2, &volume_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_floatingIPAttachGlobally(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP
	var testAccComputeV2Instance_floatingIPAttachGlobally = fmt.Sprintf(`
		resource "openstack_compute_floatingip_v2" "myip" {
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			floating_ip = "${openstack_compute_floatingip_v2.myip.address}"

			network {
				uuid = "%s"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachGlobally,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip", &fip),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_floatingIPAttachToNetwork(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP
	var testAccComputeV2Instance_floatingIPAttachToNetwork = fmt.Sprintf(`
		resource "openstack_compute_floatingip_v2" "myip" {
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]

			network {
				uuid = "%s"
				floating_ip = "${openstack_compute_floatingip_v2.myip.address}"
				access_network = true
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachToNetwork,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip", &fip),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_floatingIPAttachAndChange(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP
	var testAccComputeV2Instance_floatingIPAttachToNetwork_1 = fmt.Sprintf(`
		resource "openstack_compute_floatingip_v2" "myip_1" {
		}

		resource "openstack_compute_floatingip_v2" "myip_2" {
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]

			network {
				uuid = "%s"
				floating_ip = "${openstack_compute_floatingip_v2.myip_1.address}"
				access_network = true
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	var testAccComputeV2Instance_floatingIPAttachToNetwork_2 = fmt.Sprintf(`
		resource "openstack_compute_floatingip_v2" "myip_1" {
		}

		resource "openstack_compute_floatingip_v2" "myip_2" {
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]

			network {
				uuid = "%s"
				floating_ip = "${openstack_compute_floatingip_v2.myip_2.address}"
				access_network = true
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachToNetwork_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_1", &fip),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_floatingIPAttachToNetwork_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_2", &fip),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_multi_secgroups(t *testing.T) {
	var instance_1 servers.Server
	var secgroup_1 secgroups.SecurityGroup
	var testAccComputeV2Instance_multi_secgroups = fmt.Sprintf(`
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
			network {
				uuid = "%s"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_multi_secgroups,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_multi_secgroups_update(t *testing.T) {
	var instance_1 servers.Server
	var secgroup_1, secgroup_2 secgroups.SecurityGroup
	var testAccComputeV2Instance_multi_secgroups_update_1 = fmt.Sprintf(`
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
		}`)

	var testAccComputeV2Instance_multi_secgroups_update_2 = fmt.Sprintf(`
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
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_multi_secgroups_update_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.secgroup_2", &secgroup_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_multi_secgroups_update_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.secgroup_1", &secgroup_1),
					testAccCheckComputeV2SecGroupExists(t, "openstack_compute_secgroup_v2.secgroup_2", &secgroup_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeImage(t *testing.T) {
	var instance servers.Server
	var testAccComputeV2Instance_bootFromVolumeImage = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			block_device {
				uuid = "%s"
				source_type = "image"
				volume_size = 5
				boot_index = 0
				destination_type = "volume"
				delete_on_termination = true
			}
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeImage,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceBootVolumeAttachment(&instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeImageWithAttachedVolume(t *testing.T) {
	var instance servers.Server
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
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeImageWithAttachedVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeVolume(t *testing.T) {
	var instance servers.Server
	var testAccComputeV2Instance_bootFromVolumeVolume = fmt.Sprintf(`
	  resource "openstack_blockstorage_volume_v1" "foo" {
			name = "terraform-test"
			size = 5
			image_id = "%s"
		}

		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			block_device {
				uuid = "${openstack_blockstorage_volume_v1.foo.id}"
				source_type = "volume"
				boot_index = 0
				destination_type = "volume"
				delete_on_termination = true
			}
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceBootVolumeAttachment(&instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_bootFromVolumeForceNew(t *testing.T) {
	var instance1_1 servers.Server
	var instance1_2 servers.Server
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
		}`,
		os.Getenv("OS_IMAGE_ID"))

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
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeForceNew_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance1_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_bootFromVolumeForceNew_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance1_2),
					testAccCheckComputeV2InstanceInstanceIDsDoNotMatch(&instance1_1, &instance1_2),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_blockDeviceNewVolume(t *testing.T) {
	var instance_1 servers.Server
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
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_blockDeviceNewVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_blockDeviceExistingVolume(t *testing.T) {
	var instance_1 servers.Server
	var volume_1 volumes.Volume
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
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_blockDeviceExistingVolume,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckBlockStorageV1VolumeExists(t, "openstack_blockstorage_volume_v1.volume_1", &volume_1),
				),
			},
		},
	})
}

// TODO: verify the personality really exists on the instance.
func TestAccComputeV2Instance_personality(t *testing.T) {
	var instance servers.Server
	var testAccComputeV2Instance_personality = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			personality {
				file = "/tmp/foobar.txt"
				content = "happy"
			}
			personality {
				file = "/tmp/barfoo.txt"
				content = "angry"
			}
		}`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_personality,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_multiEphemeral(t *testing.T) {
	var instance servers.Server
	var testAccComputeV2Instance_multiEphemeral = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "foo" {
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
		}`,
		os.Getenv("OS_IMAGE_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_multiEphemeral,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_accessIPv4(t *testing.T) {
	var instance servers.Server
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
		}`, os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_accessIPv4,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					resource.TestCheckResourceAttr(
						"openstack_compute_instance_v2.instance_1", "access_ip_v4", "192.168.1.100"),
				),
			},
		},
	})
}

func TestAccComputeV2Instance_ChangeFixedIP(t *testing.T) {
	var instance1_1 servers.Server
	var instance1_2 servers.Server
	var testAccComputeV2Instance_ChangeFixedIP_1 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]
			network {
				uuid = "%s"
				fixed_ip_v4 = "10.0.0.24"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	var testAccComputeV2Instance_ChangeFixedIP_2 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]
			network {
				uuid = "%s"
				fixed_ip_v4 = "10.0.0.25"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_ChangeFixedIP_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance1_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2Instance_ChangeFixedIP_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance1_2),
					testAccCheckComputeV2InstanceInstanceIDsDoNotMatch(&instance1_1, &instance1_2),
				),
			},
		},
	})
}

func testAccCheckComputeV2InstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckComputeV2InstanceDestroy) Error creating OpenStack compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_compute_instance_v2" {
			continue
		}

		_, err := servers.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Instance still exists")
		}
	}

	return nil
}

func testAccCheckComputeV2InstanceExists(t *testing.T, n string, instance *servers.Server) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckComputeV2InstanceExists) Error creating OpenStack compute client: %s", err)
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

func testAccCheckComputeV2InstanceDoesNotExist(t *testing.T, n string, instance *servers.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckComputeV2InstanceExists) Error creating OpenStack compute client: %s", err)
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

func testAccCheckComputeV2InstanceVolumeAttachment(
	instance *servers.Server, volume *volumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var attachments []volumeattach.VolumeAttachment

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeV2Client(OS_REGION_NAME)
		if err != nil {
			return err
		}
		err = volumeattach.List(computeClient, instance.ID).EachPage(func(page pagination.Page) (bool, error) {
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
		err = volumeattach.List(computeClient, instance.ID).EachPage(func(page pagination.Page) (bool, error) {
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
		err = volumeattach.List(computeClient, instance.ID).EachPage(func(page pagination.Page) (bool, error) {
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

func TestAccComputeV2Instance_stop_before_destroy(t *testing.T) {
	var instance servers.Server
	var testAccComputeV2Instance_stop_before_destroy = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "foo" {
			name = "terraform-test"
			security_groups = ["default"]
			network {
				uuid = "%s"
			}
			stop_before_destroy = true
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2InstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Instance_stop_before_destroy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
				),
			},
		},
	})
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
		err = volumeattach.List(computeClient, instance.ID).EachPage(func(page pagination.Page) (bool, error) {
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
