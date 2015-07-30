package google

import (
	"fmt"
	"testing"

	"google.golang.org/api/compute/v1"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccInstanceGroupManager_basic(t *testing.T) {
	var manager compute.InstanceGroupManager

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManager_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-basic", &manager),
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-no-tp", &manager),
				),
			},
		},
	})
}

func TestAccInstanceGroupManager_update(t *testing.T) {
	var manager compute.InstanceGroupManager

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceGroupManager_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-update", &manager),
				),
			},
			resource.TestStep{
				Config: testAccInstanceGroupManager_update2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInstanceGroupManagerExists(
						"google_compute_instance_group_manager.igm-update", &manager),
					testAccCheckInstanceGroupManagerUpdated(
						"google_compute_instance_group_manager.igm-update", 3,
						"google_compute_target_pool.igm-update", "terraform-test-igm-update2"),
				),
			},
		},
	})
}

func testAccCheckInstanceGroupManagerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_instance_group_manager" {
			continue
		}
		_, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return fmt.Errorf("InstanceGroupManager still exists")
		}
	}

	return nil
}

func testAccCheckInstanceGroupManagerExists(n string, manager *compute.InstanceGroupManager) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("InstanceGroupManager not found")
		}

		*manager = *found

		return nil
	}
}

func testAccCheckInstanceGroupManagerUpdated(n string, size int64, targetPool string, template string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientCompute.InstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		// Cannot check the target pool as the instance creation is asynchronous.  However, can
		// check the target_size.
		if manager.TargetSize != size {
			return fmt.Errorf("instance count incorrect")
		}

		// check that the instance template updated
		instanceTemplate, err := config.clientCompute.InstanceTemplates.Get(
			config.Project, template).Do()
		if err != nil {
			return fmt.Errorf("Error reading instance template: %s", err)
		}

		if instanceTemplate.Name != template {
			return fmt.Errorf("instance template not updated")
		}

		return nil
	}
}

const testAccInstanceGroupManager_basic = `
resource "google_compute_instance_template" "igm-basic" {
	name = "terraform-test-igm-basic"
	machine_type = "n1-standard-1"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		source_image = "debian-cloud/debian-7-wheezy-v20140814"
		auto_delete = true
		boot = true
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
	}

	service_account {
		scopes = ["userinfo-email", "compute-ro", "storage-ro"]
	}
}

resource "google_compute_target_pool" "igm-basic" {
	description = "Resource created for Terraform acceptance testing"
	name = "terraform-test-igm-basic"
	session_affinity = "CLIENT_IP_PROTO"
}

resource "google_compute_instance_group_manager" "igm-basic" {
	description = "Terraform test instance group manager"
	name = "terraform-test-igm-basic"
	instance_template = "${google_compute_instance_template.igm-basic.self_link}"
	target_pools = ["${google_compute_target_pool.igm-basic.self_link}"]
	base_instance_name = "igm-basic"
	zone = "us-central1-c"
	target_size = 2
}

resource "google_compute_instance_group_manager" "igm-no-tp" {
	description = "Terraform test instance group manager"
	name = "terraform-test-igm-no-tp"
	instance_template = "${google_compute_instance_template.igm-basic.self_link}"
	base_instance_name = "igm-no-tp"
	zone = "us-central1-c"
	target_size = 2
}
`

const testAccInstanceGroupManager_update = `
resource "google_compute_instance_template" "igm-update" {
	name = "terraform-test-igm-update"
	machine_type = "n1-standard-1"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		source_image = "debian-cloud/debian-7-wheezy-v20140814"
		auto_delete = true
		boot = true
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
	}

	service_account {
		scopes = ["userinfo-email", "compute-ro", "storage-ro"]
	}
}

resource "google_compute_target_pool" "igm-update" {
	description = "Resource created for Terraform acceptance testing"
	name = "terraform-test-igm-update"
	session_affinity = "CLIENT_IP_PROTO"
}

resource "google_compute_instance_group_manager" "igm-update" {
	description = "Terraform test instance group manager"
	name = "terraform-test-igm-update"
	instance_template = "${google_compute_instance_template.igm-update.self_link}"
	target_pools = ["${google_compute_target_pool.igm-update.self_link}"]
	base_instance_name = "igm-update"
	zone = "us-central1-c"
	target_size = 2
}`

// Change IGM's instance template and target size
const testAccInstanceGroupManager_update2 = `
resource "google_compute_instance_template" "igm-update" {
	name = "terraform-test-igm-update"
	machine_type = "n1-standard-1"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		source_image = "debian-cloud/debian-7-wheezy-v20140814"
		auto_delete = true
		boot = true
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
	}

	service_account {
		scopes = ["userinfo-email", "compute-ro", "storage-ro"]
	}
}

resource "google_compute_target_pool" "igm-update" {
	description = "Resource created for Terraform acceptance testing"
	name = "terraform-test-igm-update"
	session_affinity = "CLIENT_IP_PROTO"
}

resource "google_compute_instance_template" "igm-update2" {
	name = "terraform-test-igm-update2"
	machine_type = "n1-standard-1"
	can_ip_forward = false
	tags = ["foo", "bar"]

	disk {
		source_image = "debian-cloud/debian-7-wheezy-v20140814"
		auto_delete = true
		boot = true
	}

	network_interface {
		network = "default"
	}

	metadata {
		foo = "bar"
	}

	service_account {
		scopes = ["userinfo-email", "compute-ro", "storage-ro"]
	}
}

resource "google_compute_instance_group_manager" "igm-update" {
	description = "Terraform test instance group manager"
	name = "terraform-test-igm-update"
	instance_template = "${google_compute_instance_template.igm-update2.self_link}"
	target_pools = ["${google_compute_target_pool.igm-update.self_link}"]
	base_instance_name = "igm-update"
	zone = "us-central1-c"
	target_size = 3
}`

