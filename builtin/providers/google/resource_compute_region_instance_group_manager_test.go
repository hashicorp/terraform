package google

import (
	"fmt"
	"testing"

	"google.golang.org/api/compute/v1"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccRegionInstanceGroupManager_basic(t *testing.T) {
	var manager compute.InstanceGroupManager

	template := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	target := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm2 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRegionInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRegionInstanceGroupManager_basic(template, target, igm1, igm2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegionInstanceGroupManagerExists(
						"google_compute_region_instance_group_manager.igm-region-basic", &manager),
					testAccCheckRegionInstanceGroupManagerExists(
						"google_compute_region_instance_group_manager.igm-region-no-tp", &manager),
				),
			},
		},
	})
}

func TestAccRegionInstanceGroupManager_update(t *testing.T) {
	var manager compute.InstanceGroupManager

	template1 := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	target := fmt.Sprintf("igm-test-%s", acctest.RandString(10))
	igm := fmt.Sprintf("igm-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRegionInstanceGroupManagerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRegionInstanceGroupManager_update(template1, target, igm),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegionInstanceGroupManagerExists(
						"google_compute_region_instance_group_manager.igm-region-update", &manager),
				),
			},
		},
	})
}

func testAccCheckRegionInstanceGroupManagerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_region_instance_group_manager" {
			continue
		}
		_, err := config.clientCompute.RegionInstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["region"], rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("InstanceGroupManager still exists")
		}
	}

	return nil
}

func testAccCheckRegionInstanceGroupManagerExists(n string, manager *compute.InstanceGroupManager) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.RegionInstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["region"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("RegionInstanceGroupManager not found")
		}

		*manager = *found

		return nil
	}
}

func testAccCheckRegionInstanceGroupManagerUpdated(n string, size int64, targetPool string, template string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		manager, err := config.clientCompute.RegionInstanceGroupManagers.Get(
			config.Project, rs.Primary.Attributes["region"], rs.Primary.ID).Do()
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

func testAccRegionInstanceGroupManager_basic(template, target, igm1, igm2 string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-region-basic" {
		name = "%s"
		machine_type = "f1-micro"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
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

	resource "google_compute_target_pool" "igm-region-basic" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_region_instance_group_manager" "igm-region-basic" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-region-basic.self_link}"
		target_pools = ["${google_compute_target_pool.igm-region-basic.self_link}"]
		base_instance_name = "igm-region-basic"
		region = "us-central1"
		target_size = 1
	}

	resource "google_compute_region_instance_group_manager" "igm-region-no-tp" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-region-basic.self_link}"
		base_instance_name = "igm-region-no-tp"
		region = "us-central1"
		target_size = 1
	}
	`, template, target, igm1, igm2)
}

func testAccRegionInstanceGroupManager_update(template, target, igm string) string {
	return fmt.Sprintf(`
	resource "google_compute_instance_template" "igm-region-update" {
		name = "%s"
		machine_type = "f1-micro"
		can_ip_forward = false
		tags = ["foo", "bar"]

		disk {
			source_image = "debian-cloud/debian-8-jessie-v20160803"
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

	resource "google_compute_target_pool" "igm-region-update" {
		description = "Resource created for Terraform acceptance testing"
		name = "%s"
		session_affinity = "CLIENT_IP_PROTO"
	}

	resource "google_compute_region_instance_group_manager" "igm-region-update" {
		description = "Terraform test instance group manager"
		name = "%s"
		instance_template = "${google_compute_instance_template.igm-region-update.self_link}"
		target_pools = ["${google_compute_target_pool.igm-region-update.self_link}"]
		base_instance_name = "igm-region-update"
	  region = "us-central1"
		target_size = 1
		named_port {
			name = "customhttp"
			port = 8080
		}
	}`, template, target, igm)
}
