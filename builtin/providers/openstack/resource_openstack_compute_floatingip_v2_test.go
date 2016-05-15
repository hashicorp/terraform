package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
)

func TestAccComputeV2FloatingIP_basic(t *testing.T) {
	var floatingIP floatingip.FloatingIP
	var testAccComputeV2FloatingIP_basic = `
		resource "openstack_compute_floatingip_v2" "myip_1" {
		}`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_1", &floatingIP),
				),
			},
		},
	})
}

func TestAccComputeV2FloatingIP_simpleAssociate(t *testing.T) {
	var instance servers.Server
	var fip floatingip.FloatingIP
	var testAccComputeV2FloatingIP_simpleAssociate = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]

			network {
				uuid = "%s"
			}
		}

		resource "openstack_compute_floatingip_v2" "myip" {
			instance_id = "${openstack_compute_instance_v2.instance_1.id}"
			fixed_ip = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_simpleAssociate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip", &fip),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckComputeV2FloatingIPAssociateed(&instance, &fip),
				),
			},
		},
	})
}

func TestAccComputeV2FloatingIP_associateAndChange(t *testing.T) {
	var instance_1 servers.Server
	var myip_1 floatingip.FloatingIP
	var myip_2 floatingip.FloatingIP
	var testAccComputeV2FloatingIP_associateAndChange_1 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]

			network {
				uuid = "%s"
				access_network = true
			}
		}

		resource "openstack_compute_floatingip_v2" "myip_1" {
			instance_id = "${openstack_compute_instance_v2.instance_1.id}"
			fixed_ip = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
		}

		resource "openstack_compute_floatingip_v2" "myip_2" {
			depends_on = ["openstack_compute_floatingip_v2.myip_1"]
		}`,
		os.Getenv("OS_NETWORK_ID"))

	var testAccComputeV2FloatingIP_associateAndChange_2 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]

			network {
				uuid = "%s"
				access_network = true
			}
		}

		resource "openstack_compute_floatingip_v2" "myip_1" {
		}

		resource "openstack_compute_floatingip_v2" "myip_2" {
			depends_on = ["openstack_compute_floatingip_v2.myip_1"]

			instance_id = "${openstack_compute_instance_v2.instance_1.id}"
			fixed_ip = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_associateAndChange_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_1", &myip_1),
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_2", &myip_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2FloatingIPAssociateed(&instance_1, &myip_1),
					testAccCheckComputeV2FloatingIPNotAssociateed(&instance_1, &myip_2),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_associateAndChange_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_1", &myip_1),
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_2", &myip_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2FloatingIPNotAssociateed(&instance_1, &myip_1),
					testAccCheckComputeV2FloatingIPAssociateed(&instance_1, &myip_2),
				),
			},
		},
	})
}

func TestAccComputeV2FloatingIP_associateAndDrop(t *testing.T) {
	var instance_1 servers.Server
	var myip_1 floatingip.FloatingIP
	var testAccComputeV2FloatingIP_associateAndDrop_1 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]

			network {
				uuid = "%s"
				access_network = true
			}
		}

		resource "openstack_compute_floatingip_v2" "myip_1" {
			instance_id = "${openstack_compute_instance_v2.instance_1.id}"
			fixed_ip = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
		}`,
		os.Getenv("OS_NETWORK_ID"))

	var testAccComputeV2FloatingIP_associateAndDrop_2 = fmt.Sprintf(`
		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"
			security_groups = ["default"]

			network {
				uuid = "%s"
				access_network = true
			}
		}

		resource "openstack_compute_floatingip_v2" "myip_1" {
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_associateAndDrop_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_1", &myip_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2FloatingIPAssociateed(&instance_1, &myip_1),
				),
			},
			resource.TestStep{
				Config: testAccComputeV2FloatingIP_associateAndDrop_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2FloatingIPExists(t, "openstack_compute_floatingip_v2.myip_1", &myip_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance_1),
					testAccCheckComputeV2FloatingIPNotAssociateed(&instance_1, &myip_1),
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

func testAccCheckComputeV2FloatingIPAssociateed(
	instance *servers.Server, fip *floatingip.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if fip.InstanceID == instance.ID {
			return nil
		}

		return fmt.Errorf("Floating IP %s was not associateed to instance %s", fip.ID, instance.ID)
	}
}

func testAccCheckComputeV2FloatingIPNotAssociateed(
	instance *servers.Server, fip *floatingip.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if fip.InstanceID == instance.ID {
			return fmt.Errorf("Floating IP %s is still associateed to instance %s", fip.ID, instance.ID)
		}

		return nil
	}
}
