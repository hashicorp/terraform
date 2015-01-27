package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
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
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.foo", &instance),
					testAccCheckComputeV2InstanceMetadata(&instance, "foo", "bar"),
				),
			},
		},
	})
}

func testAccCheckComputeV2InstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: OS_REGION_NAME,
	})
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

		osClient := testAccProvider.Meta().(*Config).osClient
		computeClient, err := openstack.NewComputeV2(osClient, gophercloud.EndpointOpts{
			Region: OS_REGION_NAME,
		})
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

			if v == value.(string) {
				return nil
			}

			return fmt.Errorf("Bad value for %s: %s", k, value)
		}

		return fmt.Errorf("Metadata not found: %s", k)
	}
}

var testAccComputeV2Instance_basic = fmt.Sprintf(`
	resource "openstack_compute_instance_v2" "foo" {
		region = "%s"
		name = "terraform-test"
		metadata {
			foo = "bar"
		}
	}`,
	OS_REGION_NAME)
