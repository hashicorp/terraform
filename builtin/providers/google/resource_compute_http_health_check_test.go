package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeHttpHealthCheck_basic(t *testing.T) {
	var healthCheck compute.HttpHealthCheck

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHttpHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHttpHealthCheck_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHttpHealthCheckExists(
						"google_compute_http_health_check.foobar", &healthCheck),
					testAccCheckComputeHttpHealthCheckRequestPath(
						"/health_check", &healthCheck),
					testAccCheckComputeHttpHealthCheckThresholds(
						3, 3, &healthCheck),
				),
			},
		},
	})
}

func TestAccComputeHttpHealthCheck_update(t *testing.T) {
	var healthCheck compute.HttpHealthCheck

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHttpHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHttpHealthCheck_update1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHttpHealthCheckExists(
						"google_compute_http_health_check.foobar", &healthCheck),
					testAccCheckComputeHttpHealthCheckRequestPath(
						"/not_default", &healthCheck),
					testAccCheckComputeHttpHealthCheckThresholds(
						2, 2, &healthCheck),
				),
			},
			resource.TestStep{
				Config: testAccComputeHttpHealthCheck_update2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHttpHealthCheckExists(
						"google_compute_http_health_check.foobar", &healthCheck),
					testAccCheckComputeHttpHealthCheckRequestPath(
						"/", &healthCheck),
					testAccCheckComputeHttpHealthCheckThresholds(
						10, 10, &healthCheck),
				),
			},
		},
	})
}

func testAccCheckComputeHttpHealthCheckDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_http_health_check" {
			continue
		}

		_, err := config.clientCompute.HttpHealthChecks.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("HttpHealthCheck still exists")
		}
	}

	return nil
}

func testAccCheckComputeHttpHealthCheckExists(n string, healthCheck *compute.HttpHealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.HttpHealthChecks.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("HttpHealthCheck not found")
		}

		*healthCheck = *found

		return nil
	}
}

func testAccCheckComputeHttpHealthCheckRequestPath(path string, healthCheck *compute.HttpHealthCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if healthCheck.RequestPath != path {
			return fmt.Errorf("RequestPath doesn't match: expected %s, got %s", path, healthCheck.RequestPath)
		}

		return nil
	}
}

func testAccCheckComputeHttpHealthCheckThresholds(healthy, unhealthy int64, healthCheck *compute.HttpHealthCheck) resource.TestCheckFunc {
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

var testAccComputeHttpHealthCheck_basic = fmt.Sprintf(`
resource "google_compute_http_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource created for Terraform acceptance testing"
	healthy_threshold = 3
	host = "foobar"
	name = "httphealth-test-%s"
	port = "80"
	request_path = "/health_check"
	timeout_sec = 2
	unhealthy_threshold = 3
}
`, acctest.RandString(10))

var testAccComputeHttpHealthCheck_update1 = fmt.Sprintf(`
resource "google_compute_http_health_check" "foobar" {
	name = "httphealth-test-%s"
	description = "Resource created for Terraform acceptance testing"
	request_path = "/not_default"
}
`, acctest.RandString(10))

/* Change description, restore request_path to default, and change
* thresholds from defaults */
var testAccComputeHttpHealthCheck_update2 = fmt.Sprintf(`
resource "google_compute_http_health_check" "foobar" {
	name = "httphealth-test-%s"
	description = "Resource updated for Terraform acceptance testing"
	healthy_threshold = 10
	unhealthy_threshold = 10
}
`, acctest.RandString(10))
