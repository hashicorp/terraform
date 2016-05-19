package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
)

func TestAccNetworkingV2FloatingIP_basic(t *testing.T) {
	var floatingIP floatingips.FloatingIP
	var testAccNetworkingV2FloatingIP_basic = `
		resource "openstack_networking_floatingip_v2" "fip_1" {
		}`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkingV2FloatingIP_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_1", &floatingIP),
				),
			},
		},
	})
}

func TestAccNetworkingV2FloatingIP_associate(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP
	var testAccNetworkV2FloatingIP_associate = fmt.Sprintf(`
		resource "openstack_networking_port_v2" "port_1" {
			name = "port_1"
			network_id = "%s"
			admin_state_up = "true"
		}

		resource "openstack_networking_floatingip_v2" "fip_1" {
			port_id = "${openstack_networking_port_v2.port_1.id}"
		}

		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"

			network {
				port = "${openstack_networking_port_v2.port_1.id}"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkV2FloatingIP_associate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckNetworkingV2InstanceFloatingIPAssociated(&instance, &fip),
				),
			},
		},
	})
}

func TestAccNetworkingV2FloatingIP_associateAndChange(t *testing.T) {
	var instance servers.Server
	var fip_1 floatingips.FloatingIP
	var fip_2 floatingips.FloatingIP
	var testAccNetworkV2FloatingIP_associateAndChange_1 = fmt.Sprintf(`
		resource "openstack_networking_port_v2" "port_1" {
			name = "port_1"
			network_id = "%s"
			admin_state_up = "true"
		}

		resource "openstack_networking_floatingip_v2" "fip_1" {
			port_id = "${openstack_networking_port_v2.port_1.id}"
		}

		resource "openstack_networking_floatingip_v2" "fip_2" {
			depends_on = ["openstack_networking_floatingip_v2.fip_1"]
		}

		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"

			network {
				port = "${openstack_networking_port_v2.port_1.id}"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	var testAccNetworkV2FloatingIP_associateAndChange_2 = fmt.Sprintf(`
		resource "openstack_networking_port_v2" "port_1" {
			name = "port_1"
			network_id = "%s"
			admin_state_up = "true"
		}

		resource "openstack_networking_floatingip_v2" "fip_1" {
		}

		resource "openstack_networking_floatingip_v2" "fip_2" {
			depends_on = ["openstack_networking_floatingip_v2.fip_1"]
			port_id = "${openstack_networking_port_v2.port_1.id}"
		}

		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"

			network {
				port = "${openstack_networking_port_v2.port_1.id}"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkV2FloatingIP_associateAndChange_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_1", &fip_1),
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_2", &fip_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckNetworkingV2InstanceFloatingIPAssociated(&instance, &fip_1),
					testAccCheckNetworkingV2InstanceFloatingIPNotAssociated(&instance, &fip_2),
				),
			},
			resource.TestStep{
				Config: testAccNetworkV2FloatingIP_associateAndChange_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_1", &fip_1),
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_2", &fip_2),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckNetworkingV2InstanceFloatingIPAssociated(&instance, &fip_2),
					testAccCheckNetworkingV2InstanceFloatingIPNotAssociated(&instance, &fip_1),
				),
			},
		},
	})
}

func TestAccNetworkingV2FloatingIP_associateAndDrop(t *testing.T) {
	var instance servers.Server
	var fip_1 floatingips.FloatingIP
	var testAccNetworkV2FloatingIP_associateAndDrop_1 = fmt.Sprintf(`
		resource "openstack_networking_port_v2" "port_1" {
			name = "port_1"
			network_id = "%s"
			admin_state_up = "true"
		}

		resource "openstack_networking_floatingip_v2" "fip_1" {
			port_id = "${openstack_networking_port_v2.port_1.id}"
		}

		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"

			network {
				port = "${openstack_networking_port_v2.port_1.id}"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	var testAccNetworkV2FloatingIP_associateAndDrop_2 = fmt.Sprintf(`
		resource "openstack_networking_port_v2" "port_1" {
			name = "port_1"
			network_id = "%s"
			admin_state_up = "true"
		}

		resource "openstack_networking_floatingip_v2" "fip_1" {
		}

		resource "openstack_compute_instance_v2" "instance_1" {
			name = "instance_1"

			network {
				port = "${openstack_networking_port_v2.port_1.id}"
			}
		}`,
		os.Getenv("OS_NETWORK_ID"))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkingV2FloatingIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkV2FloatingIP_associateAndDrop_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_1", &fip_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckNetworkingV2InstanceFloatingIPAssociated(&instance, &fip_1),
				),
			},
			resource.TestStep{
				Config: testAccNetworkV2FloatingIP_associateAndDrop_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkingV2FloatingIPExists(t, "openstack_networking_floatingip_v2.fip_1", &fip_1),
					testAccCheckComputeV2InstanceExists(t, "openstack_compute_instance_v2.instance_1", &instance),
					testAccCheckNetworkingV2InstanceFloatingIPNotAssociated(&instance, &fip_1),
				),
			},
		},
	})
}

func testAccCheckNetworkingV2FloatingIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	networkClient, err := config.networkingV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckNetworkingV2FloatingIPDestroy) Error creating OpenStack floating IP: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_networking_floatingip_v2" {
			continue
		}

		_, err := floatingips.Get(networkClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("FloatingIP still exists")
		}
	}

	return nil
}

func testAccCheckNetworkingV2FloatingIPExists(t *testing.T, n string, kp *floatingips.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		networkClient, err := config.networkingV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckNetworkingV2FloatingIPExists) Error creating OpenStack networking client: %s", err)
		}

		found, err := floatingips.Get(networkClient, rs.Primary.ID).Extract()
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

func testAccCheckNetworkingV2InstanceFloatingIPAssociated(
	instance *servers.Server, fip *floatingips.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, networkAddresses := range instance.Addresses {
			for _, element := range networkAddresses.([]interface{}) {
				address := element.(map[string]interface{})
				if address["OS-EXT-IPS:type"] == "fixed" && address["addr"] == fip.FixedIP {
					return nil
				}
			}
		}
		return fmt.Errorf("Floating IP %+v was not associateed to instance %+v", fip, instance)
	}
}

func testAccCheckNetworkingV2InstanceFloatingIPNotAssociated(
	instance *servers.Server, fip *floatingips.FloatingIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, networkAddresses := range instance.Addresses {
			for _, element := range networkAddresses.([]interface{}) {
				address := element.(map[string]interface{})
				if address["OS-EXT-IPS:type"] == "fixed" && address["addr"] == fip.FixedIP {
					return fmt.Errorf("Floating IP %s is still associateed to instance %s\n\nDetails: %+v\n\n%+v", fip.FloatingIP, instance.ID, fip, instance)
				}
			}
		}
		return nil
	}
}
