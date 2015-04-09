package rackspace

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	osVolumeAttach "github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
	osServers "github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
	rsVolumes "github.com/rackspace/gophercloud/rackspace/blockstorage/v1/volumes"
	rsServers "github.com/rackspace/gophercloud/rackspace/compute/v2/servers"
	rsVolumeAttach "github.com/rackspace/gophercloud/rackspace/compute/v2/volumeattach"
)

func TestAccComputeInstance_basic(t *testing.T) {
	var instance osServers.Server
	var testAccComputeInstance_basic = fmt.Sprintf(`
      resource "rackspace_compute_instance" "foo" {
        name = "terraform-test"
        network {
          uuid = "%s"
        }
        metadata {
          foo = "bar"
        }
        }`,
		os.Getenv("RS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(t, "rackspace_compute_instance.foo", &instance),
					testAccCheckComputeInstanceMetadata(&instance, "foo", "bar"),
				),
			},
		},
	})
}

func TestAccComputeInstance_volumeAttach(t *testing.T) {
	var instance osServers.Server
	var volume rsVolumes.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_volumeAttach,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBlockStorageVolumeExists(t, "rackspace_blockstorage_volume.myvol", &volume),
					testAccCheckComputeInstanceExists(t, "rackspace_compute_instance.foo", &instance),
					testAccCheckComputeInstanceVolumeAttachment(&instance, &volume),
				),
			},
		},
	})
}

func testAccCheckComputeInstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeClient(RS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckComputeInstanceDestroy) Error creating Rackspace compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rackspace_compute_instance" {
			continue
		}

		_, err := rsServers.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Instance still exists")
		}
	}

	return nil
}

func testAccCheckComputeInstanceExists(t *testing.T, n string, instance *osServers.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeClient(RS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckComputeInstanceExists) Error creating Rackspace compute client: %s", err)
		}

		found, err := rsServers.Get(computeClient, rs.Primary.ID).Extract()
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

func testAccCheckComputeInstanceMetadata(
	instance *osServers.Server, k string, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if instance.Metadata == nil {
			return fmt.Errorf("No metadata")
		}

		for key, value := range instance.Metadata {
			if k != key {
				continue
			}

			if v == value.(string) {
				return nil
			}

			return fmt.Errorf("Bad value for %s: %s", k, value)
		}

		return fmt.Errorf("Metadata not found: %s", k)
	}
}

func testAccCheckComputeInstanceVolumeAttachment(
	instance *osServers.Server, volume *rsVolumes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var attachments []osVolumeAttach.VolumeAttachment

		config := testAccProvider.Meta().(*Config)
		computeClient, err := config.computeClient(RS_REGION_NAME)
		if err != nil {
			return err
		}
		err = rsVolumeAttach.List(computeClient, instance.ID).EachPage(func(page pagination.Page) (bool, error) {
			actual, err := osVolumeAttach.ExtractVolumeAttachments(page)
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

var testAccComputeInstance_volumeAttach = fmt.Sprintf(`
                                            resource "rackspace_blockstorage_volume" "myvol" {
                                              name = "myvol"
                                              size = 1
                                            }

                                            resource "rackspace_compute_instance" "foo" {
                                              region = "%s"
                                              name = "terraform-test"
                                              volume {
                                                volume_id = "${rackspace_blockstorage_volume.myvol.id}"
                                              }
                                              }`,
	RS_REGION_NAME)
