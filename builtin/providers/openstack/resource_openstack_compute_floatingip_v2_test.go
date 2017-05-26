package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

func TestAccComputeV2FloatingIP_basic(t *testing.T) {
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists("openstack_compute_floatingip_v2.fip_1", &fip),
				),
			},
		},
	})
}

func TestAccComputeV2FloatingIP_attach(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_attach,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists("openstack_compute_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2InstanceExists("openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2InstanceFloatingIPAttach(&instance, &fip),
				),
			},
		},
	})
}

func testAccCheckComputeV2FloatingIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_compute_floatingip_v2" {
			continue
		}

		_, err := floatingips.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("FloatingIP still exists")
		}
	}

	return nil
}

func testAccCheckComputeV2FloatingIPExists(n string, kp *floatingips.FloatingIP) resource.TestCheckFunc {
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

		found, err := floatingips.Get(computeClient, rs.Primary.ID).Extract()
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

const testAccComputeV2FloatingIP_basic = `
resource "openstack_compute_floatingip_v2" "fip_1" {
}
`

var testAccComputeV2FloatingIP_attach = fmt.Sprintf(`
resource "openstack_compute_floatingip_v2" "fip_1" {
}

resource "openstack_compute_instance_v2" "instance_1" {
	name = "instance_1"
	security_groups = ["default"]
	floating_ip = "${openstack_compute_floatingip_v2.fip_1.address}"

	network {
		uuid = "%s"
	}
}
`, OS_NETWORK_ID)
