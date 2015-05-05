package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
)

func TestAccComputeV2ServerGroup_basic(t *testing.T) {
	var serverGroup servergroups.ServerGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2ServerGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2ServerGroup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2ServerGroupExists(t, "openstack_compute_servergroup_v2.mysg", &serverGroup),
				),
			},
		},
	})
}

func TestAccComputeV2ServerGroup_affinity(t *testing.T) {
	var instance servers.Server
	var sg servergroups.ServerGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2ServerGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2ServerGroup_affinity,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2ServerGroupExists(t, "openstack_compute_servergroup_v2.mysg", &sg),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.myinstance", &instance),
					testAccCheckComputeV2InstanceInServerGroup(&instance, &sg),
				),
			},
		},
	})
}

func testAccCheckComputeV2ServerGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckComputeV2ServerGroupDestroy) Error creating OpenStack compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_compute_servergroup_v2" {
			continue
		}

		_, err := servergroups.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("ServerGroup still exists")
		}
	}

	return nil
}

func testAccCheckComputeV2ServerGroupExists(t *testing.T, n string, kp *servergroups.ServerGroup) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckComputeV2ServerGroupExists) Error creating OpenStack compute client: %s", err)
		}

		found, err := servergroups.Get(computeClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("ServerGroup not found")
		}

		*kp = *found

		return nil
	}
}

func testAccCheckComputeV2InstanceInServerGroup(instance *servers.Server, sg *servergroups.ServerGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(sg.Members) > 0 {
			for _, m := range sg.Members {
				if m == instance.ID {
					return nil
				}
			}
		}

		return fmt.Errorf("Instance %s is not part of Server Group %s", instance.ID, sg.ID)
	}
}

var testAccComputeV2ServerGroup_basic = `
  resource "openstack_compute_servergroup_v2" "mysg" {
		name = "mysg"
		policies = ["affinity"]
  }`

var testAccComputeV2ServerGroup_affinity = `
	resource "openstack_compute_servergroup_v2" "mysg" {
		name = "mysg"
		policies = ["affinity"]
	}

	resource "openstack_compute_instance_v2" "myinstance" {
		name = "myinstance"
		security_groups = ["default"]
		scheduler_hints {
			group = "${openstack_compute_servergroup_v2.mysg.id}"
		}
	}`
