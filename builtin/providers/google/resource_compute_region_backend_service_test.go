package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeRegionBackendService_basic(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	extraCheckName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRegionBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRegionBackendService_basic(serviceName, checkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRegionBackendServiceExists(
						"google_compute_region_backend_service.foobar", &svc),
				),
			},
			resource.TestStep{
				Config: testAccComputeRegionBackendService_basicModified(
					serviceName, checkName, extraCheckName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRegionBackendServiceExists(
						"google_compute_region_backend_service.foobar", &svc),
				),
			},
		},
	})
}

func TestAccComputeRegionBackendService_withBackend(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	igName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	itName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRegionBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRegionBackendService_withBackend(
					serviceName, igName, itName, checkName, 10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRegionBackendServiceExists(
						"google_compute_region_backend_service.lipsum", &svc),
				),
			},
		},
	})

	if svc.TimeoutSec != 10 {
		t.Errorf("Expected TimeoutSec == 10, got %d", svc.TimeoutSec)
	}
	if svc.Protocol != "TCP" {
		t.Errorf("Expected Protocol to be TCP, got %q", svc.Protocol)
	}
	if len(svc.Backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(svc.Backends))
	}
}

func TestAccComputeRegionBackendService_withBackendAndUpdate(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	igName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	itName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRegionBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRegionBackendService_withBackend(
					serviceName, igName, itName, checkName, 10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRegionBackendServiceExists(
						"google_compute_region_backend_service.lipsum", &svc),
				),
			},
			resource.TestStep{
				Config: testAccComputeRegionBackendService_withBackend(
					serviceName, igName, itName, checkName, 20),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRegionBackendServiceExists(
						"google_compute_region_backend_service.lipsum", &svc),
				),
			},
		},
	})

	if svc.TimeoutSec != 20 {
		t.Errorf("Expected TimeoutSec == 20, got %d", svc.TimeoutSec)
	}
	if svc.Protocol != "TCP" {
		t.Errorf("Expected Protocol to be TCP, got %q", svc.Protocol)
	}
	if len(svc.Backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(svc.Backends))
	}
}

func TestAccComputeRegionBackendService_withSessionAffinity(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRegionBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRegionBackendService_withSessionAffinity(
					serviceName, checkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeRegionBackendServiceExists(
						"google_compute_region_backend_service.foobar", &svc),
				),
			},
		},
	})

	if svc.SessionAffinity != "CLIENT_IP" {
		t.Errorf("Expected Protocol to be CLIENT_IP, got %q", svc.SessionAffinity)
	}
}

func testAccCheckComputeRegionBackendServiceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_region_backend_service" {
			continue
		}

		_, err := config.clientCompute.RegionBackendServices.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Backend service still exists")
		}
	}

	return nil
}

func testAccCheckComputeRegionBackendServiceExists(n string, svc *compute.BackendService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.RegionBackendServices.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Backend service not found")
		}

		*svc = *found

		return nil
	}
}

func testAccComputeRegionBackendService_basic(serviceName, checkName string) string {
	return fmt.Sprintf(`
resource "google_compute_region_backend_service" "foobar" {
  name                  = "%s"
  health_checks         = ["${google_compute_health_check.zero.self_link}"]
  region                = "us-central1"
}

resource "google_compute_health_check" "zero" {
  name               = "%s"
  check_interval_sec = 1
  timeout_sec        = 1

  tcp_health_check {
    port = "80"
  }
}
`, serviceName, checkName)
}

func testAccComputeRegionBackendService_basicModified(serviceName, checkOne, checkTwo string) string {
	return fmt.Sprintf(`
resource "google_compute_region_backend_service" "foobar" {
    name = "%s"
    health_checks = ["${google_compute_health_check.one.self_link}"]
    region = "us-central1"
}

resource "google_compute_health_check" "zero" {
    name = "%s"
    check_interval_sec = 1
    timeout_sec = 1

    tcp_health_check {
    }
}

resource "google_compute_health_check" "one" {
    name = "%s"
    check_interval_sec = 30
    timeout_sec = 30

    tcp_health_check {
    }
}
`, serviceName, checkOne, checkTwo)
}

func testAccComputeRegionBackendService_withBackend(
	serviceName, igName, itName, checkName string, timeout int64) string {
	return fmt.Sprintf(`
resource "google_compute_region_backend_service" "lipsum" {
  name        = "%s"
  description = "Hello World 1234"
  protocol    = "TCP"
  region      = "us-central1"
  timeout_sec = %v

  backend {
    group = "${google_compute_instance_group_manager.foobar.instance_group}"
  }

  health_checks = ["${google_compute_health_check.default.self_link}"]
}

resource "google_compute_instance_group_manager" "foobar" {
  name               = "%s"
  instance_template  = "${google_compute_instance_template.foobar.self_link}"
  base_instance_name = "foobar"
  zone               = "us-central1-f"
  target_size        = 1
}

resource "google_compute_instance_template" "foobar" {
  name         = "%s"
  machine_type = "n1-standard-1"

  network_interface {
    network = "default"
  }

  disk {
    source_image = "debian-8-jessie-v20160803"
    auto_delete  = true
    boot         = true
  }
}

resource "google_compute_health_check" "default" {
  name               = "%s"
  check_interval_sec = 1
  timeout_sec        = 1

  tcp_health_check {

  }
}
`, serviceName, timeout, igName, itName, checkName)
}

func testAccComputeRegionBackendService_withSessionAffinity(serviceName, checkName string) string {
	return fmt.Sprintf(`
resource "google_compute_region_backend_service" "foobar" {
  name                  = "%s"
  health_checks         = ["${google_compute_health_check.zero.self_link}"]
  region                = "us-central1"
  session_affinity      = "CLIENT_IP"

}

resource "google_compute_health_check" "zero" {
  name               = "%s"
  check_interval_sec = 1
  timeout_sec        = 1

  tcp_health_check {
    port = "80"
  }
}
`, serviceName, checkName)
}
