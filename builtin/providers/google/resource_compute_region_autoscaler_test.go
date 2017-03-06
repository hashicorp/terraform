package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccRegionAutoscaler_basic(t *testing.T) {
	var regascaler compute.Autoscaler

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRegionAutoscalerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRegionAutoscaler_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegionAutoscalerExists(
						"google_compute_region_autoscaler.foobar", &regascaler),
				),
			},
		},
	})
}

func TestAccRegionAutoscaler_update(t *testing.T) {
	var regascaler compute.Autoscaler

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRegionAutoscalerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRegionAutoscaler_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegionAutoscalerExists(
						"google_compute_region_autoscaler.foobar", &regascaler),
				),
			},
			resource.TestStep{
				Config: testAccRegionAutoscaler_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegionAutoscalerExists(
						"google_compute_region_autoscaler.foobar", &regascaler),
					testAccCheckRegionAutoscalerUpdated(
						"google_compute_region_autoscaler.foobar", 10),
				),
			},
		},
	})
}

func testAccCheckRegionAutoscalerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_region_autoscaler" {
			continue
		}

		_, err := config.clientCompute.RegionAutoscalers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("RegionAutoscaler still exists")
		}
	}

	return nil
}

func testAccCheckRegionAutoscalerExists(n string, regascaler *compute.Autoscaler) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.RegionAutoscalers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("RegionAutoscaler not found")
		}

		*regascaler = *found

		return nil
	}
}

func testAccCheckRegionAutoscalerUpdated(n string, max int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		regascaler, err := config.clientCompute.RegionAutoscalers.Get(
			config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if regascaler.AutoscalingPolicy.MaxNumReplicas != max {
			return fmt.Errorf("maximum replicas incorrect")
		}

		return nil
	}
}

var testAccRegionAutoscaler_basic = fmt.Sprintf(`
resource "google_compute_instance_template" "foobar" {
	name = "regascaler-test-%s"
	machine_type = "n1-standard-1"
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

resource "google_compute_target_pool" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "regascaler-test-%s"
	session_affinity = "CLIENT_IP_PROTO"
}

resource "google_compute_region_instance_group_manager" "foobar" {
	description = "Terraform test instance group manager"
	name = "regascaler-test-%s"
	instance_template = "${google_compute_instance_template.foobar.self_link}"
	target_pools = ["${google_compute_target_pool.foobar.self_link}"]
	base_instance_name = "foobar"
	region = "us-central1"
}

resource "google_compute_region_autoscaler" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "regascaler-test-%s"
	region = "us-central1"
	target = "${google_compute_region_instance_group_manager.foobar.self_link}"
	autoscaling_policy = {
		max_replicas = 5
		min_replicas = 1
		cooldown_period = 60
		cpu_utilization = {
			target = 0.5
		}
	}

}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))

var testAccRegionAutoscaler_update = fmt.Sprintf(`
resource "google_compute_instance_template" "foobar" {
	name = "regascaler-test-%s"
	machine_type = "n1-standard-1"
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

resource "google_compute_target_pool" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "regascaler-test-%s"
	session_affinity = "CLIENT_IP_PROTO"
}

resource "google_compute_region_instance_group_manager" "foobar" {
	description = "Terraform test instance group manager"
	name = "regascaler-test-%s"
	instance_template = "${google_compute_instance_template.foobar.self_link}"
	target_pools = ["${google_compute_target_pool.foobar.self_link}"]
	base_instance_name = "foobar"
	region = "us-central1"
}

resource "google_compute_region_autoscaler" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	name = "regascaler-test-%s"
	region = "us-central1"
	target = "${google_compute_region_instance_group_manager.foobar.self_link}"
	autoscaling_policy = {
		max_replicas = 10
		min_replicas = 1
		cooldown_period = 60
		cpu_utilization = {
			target = 0.5
		}
	}

}`, acctest.RandString(10), acctest.RandString(10), acctest.RandString(10), acctest.RandString(10))
