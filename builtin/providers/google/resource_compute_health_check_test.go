package google

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeHealthCheck_tcp(t *testing.T) {
	var healthCheck compute.HealthCheck

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHealthCheck_tcp,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHealthCheckExists(
						"google_compute_health_check.foobar", &healthCheck),
					testAccCheckComputeHealthCheckThresholds(
						3, 3, &healthCheck),
					testAccCheckComputeHealthCheckTcpPort(80, &healthCheck),
				),
			},
		},
	})
}

func TestAccComputeHealthCheck_tcp_update(t *testing.T) {
	var healthCheck compute.HealthCheck

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHealthCheck_tcp,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHealthCheckExists(
						"google_compute_health_check.foobar", &healthCheck),
					testAccCheckComputeHealthCheckThresholds(
						3, 3, &healthCheck),
					testAccCheckComputeHealthCheckTcpPort(80, &healthCheck),
				),
			},
			resource.TestStep{
				Config: testAccComputeHealthCheck_tcp_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHealthCheckExists(
						"google_compute_health_check.foobar", &healthCheck),
					testAccCheckComputeHealthCheckThresholds(
						10, 10, &healthCheck),
					testAccCheckComputeHealthCheckTcpPort(8080, &healthCheck),
				),
			},
		},
	})
}

func TestAccComputeHealthCheck_ssl(t *testing.T) {
	var healthCheck compute.HealthCheck

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHealthCheck_ssl,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHealthCheckExists(
						"google_compute_health_check.foobar", &healthCheck),
					testAccCheckComputeHealthCheckThresholds(
						3, 3, &healthCheck),
				),
			},
		},
	})
}

func TestAccComputeHealthCheck_http(t *testing.T) {
	var healthCheck compute.HealthCheck

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHealthCheck_http,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHealthCheckExists(
						"google_compute_health_check.foobar", &healthCheck),
					testAccCheckComputeHealthCheckThresholds(
						3, 3, &healthCheck),
				),
			},
		},
	})
}

func TestAccComputeHealthCheck_https(t *testing.T) {
	var healthCheck compute.HealthCheck

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHealthCheck_https,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHealthCheckExists(
						"google_compute_health_check.foobar", &healthCheck),
					testAccCheckComputeHealthCheckThresholds(
						3, 3, &healthCheck),
				),
			},
		},
	})
}

func TestAccComputeHealthCheck_tcpAndSsl_shouldFail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:      testAccComputeHealthCheck_tcpAndSsl_shouldFail,
				ExpectError: regexp.MustCompile("conflicts with tcp_health_check"),
			},
		},
	})
}

func testAccCheckComputeHealthCheckDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_health_check" {
			continue
		}

		_, err := config.clientCompute.HealthChecks.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("HealthCheck %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckComputeHealthCheckExists(n string, healthCheck *compute.HealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.HealthChecks.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("HealthCheck not found")
		}

		*healthCheck = *found

		return nil
	}
}

func testAccCheckErrorCreating(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if ok {
			return fmt.Errorf("HealthCheck %s created successfully with bad config", n)
		}
		return nil
	}
}

func testAccCheckComputeHealthCheckThresholds(healthy, unhealthy int64, healthCheck *compute.HealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if healthCheck.HealthyThreshold != healthy {
			return fmt.Errorf("HealthyThreshold doesn't match: expected %d, got %d", healthy, healthCheck.HealthyThreshold)
		}

		if healthCheck.UnhealthyThreshold != unhealthy {
			return fmt.Errorf("UnhealthyThreshold doesn't match: expected %d, got %d", unhealthy, healthCheck.UnhealthyThreshold)
		}

		return nil
	}
}

func testAccCheckComputeHealthCheckTcpPort(port int64, healthCheck *compute.HealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if healthCheck.TcpHealthCheck.Port != port {
			return fmt.Errorf("Port doesn't match: expected %v, got %v", port, healthCheck.TcpHealthCheck.Port)
		}
		return nil
	}
}

var testAccComputeHealthCheck_tcp = fmt.Sprintf(`
resource "google_compute_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource created for Terraform acceptance testing"
	healthy_threshold = 3
	name = "health-test-%s"
	timeout_sec = 2
	unhealthy_threshold = 3
	tcp_health_check {
	}
}
`, acctest.RandString(10))

var testAccComputeHealthCheck_tcp_update = fmt.Sprintf(`
resource "google_compute_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource updated for Terraform acceptance testing"
	healthy_threshold = 10
	name = "health-test-%s"
	timeout_sec = 2
	unhealthy_threshold = 10
	tcp_health_check {
		port = "8080"
	}
}
`, acctest.RandString(10))

var testAccComputeHealthCheck_ssl = fmt.Sprintf(`
resource "google_compute_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource created for Terraform acceptance testing"
	healthy_threshold = 3
	name = "health-test-%s"
	timeout_sec = 2
	unhealthy_threshold = 3
	ssl_health_check {
		port = "443"
	}
}
`, acctest.RandString(10))

var testAccComputeHealthCheck_http = fmt.Sprintf(`
resource "google_compute_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource created for Terraform acceptance testing"
	healthy_threshold = 3
	name = "health-test-%s"
	timeout_sec = 2
	unhealthy_threshold = 3
	http_health_check {
		port = "80"
	}
}
`, acctest.RandString(10))

var testAccComputeHealthCheck_https = fmt.Sprintf(`
resource "google_compute_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource created for Terraform acceptance testing"
	healthy_threshold = 3
	name = "health-test-%s"
	timeout_sec = 2
	unhealthy_threshold = 3
	https_health_check {
		port = "443"
	}
}
`, acctest.RandString(10))

var testAccComputeHealthCheck_tcpAndSsl_shouldFail = fmt.Sprintf(`
resource "google_compute_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource created for Terraform acceptance testing"
	healthy_threshold = 3
	name = "health-test-%s"
	timeout_sec = 2
	unhealthy_threshold = 3

	tcp_health_check {
	}
	ssl_health_check {
	}
}
`, acctest.RandString(10))
