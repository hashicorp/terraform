package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
)

func TestAccComputeV2FloatingIP_basic(t *testing.T) {
	var floatingIP floatingip.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.foo", &floatingIP),
				),
			},
		},
	})
}

func testAccCheckComputeV2FloatingIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckComputeV2FloatingIPDestroy) Error creating OpenStack compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_compute_floatingip_v2" {
			continue
		}

		_, err := floatingip.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("FloatingIP still exists")
		}
	}

	return nil
}

func testAccCheckComputeV2FloatingIPExists(t *testing.T, n string, kp *floatingip.FloatingIP) resource.TestCheckFunc {
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
			return fmt.Errorf("(testAccCheckComputeV2FloatingIPExists) Error creating OpenStack compute client: %s", err)
		}

		found, err := floatingip.Get(computeClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("FloatingIP not found")
		}

		*kp = *found

		return nil
	}
}

var testAccComputeV2FloatingIP_basic = `
  resource "openstack_compute_floatingip_v2" "foo" {
  }

  resource "openstack_compute_instance_v2" "bar" {
	name = "terraform-acc-floating-ip-test"
	floating_ip = "${openstack_compute_floatingip_v2.foo.address}"
  }`
