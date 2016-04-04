package google

import (
	"fmt"
	"testing"

	"google.golang.org/api/compute/v1"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeInstanceGroup_basic(t *testing.T) {
	var instanceGroup compute.InstanceGroup
	var instanceName = fmt.Sprintf("instancegroup-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccComputeInstanceGroup_destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstanceGroup_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccComputeInstanceGroup_exists(
						"google_compute_instance_group.basic", &instanceGroup),
					testAccComputeInstanceGroup_exists(
						"google_compute_instance_group.empty", &instanceGroup),
				),
			},
		},
	})
}

func TestAccComputeInstanceGroup_update(t *testing.T) {
	var instanceGroup compute.InstanceGroup
	var instanceName = fmt.Sprintf("instancegroup-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccComputeInstanceGroup_destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstanceGroup_update(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccComputeInstanceGroup_exists(
						"google_compute_instance_group.update", &instanceGroup),
					testAccComputeInstanceGroup_named_ports(
						"google_compute_instance_group.update",
						map[string]int64{"http": 8080, "https": 8443},
						&instanceGroup),
				),
			},
			resource.TestStep{
				Config: testAccComputeInstanceGroup_update2(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccComputeInstanceGroup_exists(
						"google_compute_instance_group.update", &instanceGroup),
					testAccComputeInstanceGroup_updated(
						"google_compute_instance_group.update", 3, &instanceGroup),
					testAccComputeInstanceGroup_named_ports(
						"google_compute_instance_group.update",
						map[string]int64{"http": 8081, "test": 8444},
						&instanceGroup),
				),
			},
		},
	})
}

func testAccComputeInstanceGroup_destroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_instance_group" {
			continue
		}
		_, err := config.clientCompute.InstanceGroups.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("InstanceGroup still exists")
		}
	}

	return nil
}

func testAccComputeInstanceGroup_exists(n string, instanceGroup *compute.InstanceGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.InstanceGroups.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("InstanceGroup not found")
		}

		*instanceGroup = *found

		return nil
	}
}

func testAccComputeInstanceGroup_updated(n string, size int64, instanceGroup *compute.InstanceGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		instanceGroup, err := config.clientCompute.InstanceGroups.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		// Cannot check the target pool as the instance creation is asynchronous.  However, can
		// check the target_size.
		if instanceGroup.Size != size {
			return fmt.Errorf("instance count incorrect")
		}

		return nil
	}
}

func testAccComputeInstanceGroup_named_ports(n string, np map[string]int64, instanceGroup *compute.InstanceGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		instanceGroup, err := config.clientCompute.InstanceGroups.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		var found bool
		for _, namedPort := range instanceGroup.NamedPorts {
			found = false
			for name, port := range np {
				if namedPort.Name == name && namedPort.Port == port {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("named port incorrect")
			}
		}

		return nil
	}
}

func testAccComputeInstanceGroup_basic(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "ig_instance" {
		name = "%s"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		zone = "us-central1-c"

		disk {
			image = "debian-7-wheezy-v20160301"
		}

		network_interface {
			network = "default"
		}
	}

	resource "google_compute_instance_group" "basic" {
		description = "Terraform test instance group"
		name = "%s"
		zone = "us-central1-c"
		instances = [ "${google_compute_instance.ig_instance.self_link}" ]
		named_port {
			name = "http"
			port = "8080"
		}
		named_port {
			name = "https"
			port = "8443"
		}
	}

	resource "google_compute_instance_group" "empty" {
		description = "Terraform test instance group empty"
		name = "%s-empty"
		zone = "us-central1-c"
			named_port {
			name = "http"
			port = "8080"
		}
		named_port {
			name = "https"
			port = "8443"
		}
	}`, instance, instance, instance)
}

func testAccComputeInstanceGroup_update(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "ig_instance" {
		name = "%s-${count.index}"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		zone = "us-central1-c"
		count = 1

		disk {
			image = "debian-7-wheezy-v20160301"
		}

		network_interface {
			network = "default"
		}
	}

	resource "google_compute_instance_group" "update" {
		description = "Terraform test instance group"
		name = "%s"
		zone = "us-central1-c"
		instances = [ "${google_compute_instance.ig_instance.self_link}" ]
		named_port {
			name = "http"
			port = "8080"
		}
		named_port {
			name = "https"
			port = "8443"
		}
	}`, instance, instance)
}

// Change IGM's instance template and target size
func testAccComputeInstanceGroup_update2(instance string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance" "ig_instance" {
		name = "%s-${count.index}"
		machine_type = "n1-standard-1"
		can_ip_forward = false
		zone = "us-central1-c"
		count = 3

		disk {
			image = "debian-7-wheezy-v20160301"
		}

		network_interface {
			network = "default"
		}
	}

	resource "google_compute_instance_group" "update" {
		description = "Terraform test instance group"
		name = "%s"
		zone = "us-central1-c"
		instances = [ "${google_compute_instance.ig_instance.*.self_link}" ]

		named_port {
			name = "http"
			port = "8081"
		}
		named_port {
			name = "test"
			port = "8444"
		}
	}`, instance, instance)
}
