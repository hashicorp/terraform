package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeBackendService_basic(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	extraCheckName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendService_basic(serviceName, checkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
			resource.TestStep{
				Config: testAccComputeBackendService_basicModified(
					serviceName, checkName, extraCheckName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
		},
	})
}

func TestAccComputeBackendService_withBackend(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	igName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	itName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendService_withBackend(
					serviceName, igName, itName, checkName, 10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.lipsum", &svc),
				),
			},
		},
	})

	if svc.TimeoutSec != 10 {
		t.Errorf("Expected TimeoutSec == 10, got %d", svc.TimeoutSec)
	}
	if svc.Protocol != "HTTP" {
		t.Errorf("Expected Protocol to be HTTP, got %q", svc.Protocol)
	}
	if len(svc.Backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(svc.Backends))
	}
}

func TestAccComputeBackendService_withBackendAndUpdate(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	igName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	itName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendService_withBackend(
					serviceName, igName, itName, checkName, 10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.lipsum", &svc),
				),
			},
			resource.TestStep{
				Config: testAccComputeBackendService_withBackend(
					serviceName, igName, itName, checkName, 20),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.lipsum", &svc),
				),
			},
		},
	})

	if svc.TimeoutSec != 20 {
		t.Errorf("Expected TimeoutSec == 20, got %d", svc.TimeoutSec)
	}
	if svc.Protocol != "HTTP" {
		t.Errorf("Expected Protocol to be HTTP, got %q", svc.Protocol)
	}
	if len(svc.Backends) != 1 {
		t.Errorf("Expected 1 backend, got %d", len(svc.Backends))
	}
}

func TestAccComputeBackendService_withConnectionDraining(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendService_withConnectionDraining(serviceName, checkName, 10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
		},
	})

	if svc.ConnectionDraining.DrainingTimeoutSec != 10 {
		t.Errorf("Expected ConnectionDraining.DrainingTimeoutSec == 10, got %d", svc.ConnectionDraining.DrainingTimeoutSec)
	}
}

func TestAccComputeBackendService_withConnectionDrainingAndUpdate(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendService_withConnectionDraining(serviceName, checkName, 10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
			resource.TestStep{
				Config: testAccComputeBackendService_basic(serviceName, checkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
		},
	})

	if svc.ConnectionDraining.DrainingTimeoutSec != 0 {
		t.Errorf("Expected ConnectionDraining.DrainingTimeoutSec == 0, got %d", svc.ConnectionDraining.DrainingTimeoutSec)
	}
}

func testAccCheckComputeBackendServiceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_backend_service" {
			continue
		}

		_, err := config.clientCompute.BackendServices.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Backend service %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckComputeBackendServiceExists(n string, svc *compute.BackendService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.BackendServices.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Backend service %s not found", rs.Primary.ID)
		}

		*svc = *found

		return nil
	}
}

func TestAccComputeBackendService_withCDNEnabled(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendService_withCDNEnabled(
					serviceName, checkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
		},
	})

	if svc.EnableCDN != true {
		t.Errorf("Expected EnableCDN == true, got %t", svc.EnableCDN)
	}
}

func TestAccComputeBackendService_withSessionAffinity(t *testing.T) {
	serviceName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	checkName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var svc compute.BackendService

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeBackendServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeBackendService_withSessionAffinity(
					serviceName, checkName, "CLIENT_IP"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
			resource.TestStep{
				Config: testAccComputeBackendService_withSessionAffinity(
					serviceName, checkName, "GENERATED_COOKIE"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeBackendServiceExists(
						"google_compute_backend_service.foobar", &svc),
				),
			},
		},
	})

	if svc.SessionAffinity != "GENERATED_COOKIE" {
		t.Errorf("Expected SessionAffinity == \"GENERATED_COOKIE\", got %s", svc.SessionAffinity)
	}
}

func testAccComputeBackendService_basic(serviceName, checkName string) string {
	return fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
  name          = "%s"
  health_checks = ["${google_compute_http_health_check.zero.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
  name               = "%s"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}
`, serviceName, checkName)
}

func testAccComputeBackendService_withCDNEnabled(serviceName, checkName string) string {
	return fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
  name          = "%s"
  health_checks = ["${google_compute_http_health_check.zero.self_link}"]
  enable_cdn    = true
}

resource "google_compute_http_health_check" "zero" {
  name               = "%s"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}
`, serviceName, checkName)
}

func testAccComputeBackendService_basicModified(serviceName, checkOne, checkTwo string) string {
	return fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
    name = "%s"
    health_checks = ["${google_compute_http_health_check.one.self_link}"]
}

resource "google_compute_http_health_check" "zero" {
    name = "%s"
    request_path = "/"
    check_interval_sec = 1
    timeout_sec = 1
}

resource "google_compute_http_health_check" "one" {
    name = "%s"
    request_path = "/one"
    check_interval_sec = 30
    timeout_sec = 30
}
`, serviceName, checkOne, checkTwo)
}

func testAccComputeBackendService_withBackend(
	serviceName, igName, itName, checkName string, timeout int64) string {
	return fmt.Sprintf(`
resource "google_compute_backend_service" "lipsum" {
  name        = "%s"
  description = "Hello World 1234"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = %v

  backend {
    group = "${google_compute_instance_group_manager.foobar.instance_group}"
  }

  health_checks = ["${google_compute_http_health_check.default.self_link}"]
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

resource "google_compute_http_health_check" "default" {
  name               = "%s"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}
`, serviceName, timeout, igName, itName, checkName)
}

func testAccComputeBackendService_withSessionAffinity(serviceName, checkName, affinityName string) string {
	return fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
  name             = "%s"
  health_checks    = ["${google_compute_http_health_check.zero.self_link}"]
  session_affinity = "%s"
}

resource "google_compute_http_health_check" "zero" {
  name               = "%s"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}
`, serviceName, affinityName, checkName)
}

func testAccComputeBackendService_withConnectionDraining(serviceName, checkName string, drainingTimeout int64) string {
	return fmt.Sprintf(`
resource "google_compute_backend_service" "foobar" {
  name          = "%s"
  health_checks = ["${google_compute_http_health_check.zero.self_link}"]
  connection_draining_timeout_sec = %v
}

resource "google_compute_http_health_check" "zero" {
  name               = "%s"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}
`, serviceName, drainingTimeout, checkName)
}
