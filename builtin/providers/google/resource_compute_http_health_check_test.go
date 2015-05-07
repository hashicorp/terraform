package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeHttpHealthCheck_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHttpHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHttpHealthCheck_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHttpHealthCheckExists(
						"google_compute_http_health_check.foobar"),
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
				),
			},
			resource.TestStep{
				Config: testAccComputeHttpHealthCheck_update2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeHttpHealthCheckExists(
						"google_compute_http_health_check.foobar", &healthCheck),
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

func testAccCheckComputeHttpHealthCheckExists(n string) resource.TestCheckFunc {
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

		return nil
	}
}

const testAccComputeHttpHealthCheck_basic = `
resource "google_compute_http_health_check" "foobar" {
	check_interval_sec = 3
	description = "Resource created for Terraform acceptance testing"
	healthy_threshold = 3
	host = "foobar"
	name = "terraform-test"
	port = "80"
	request_path = "/health_check"
	timeout_sec = 2
	unhealthy_threshold = 3
}
`

const testAccComputeHttpHealthCheck_update1 = `
resource "google_compute_http_health_check" "foobar" {
	name = "terraform-test"
	description = "Resource created for Terraform acceptance testing"
	request_path = "/not_default"
}
`

/* Change description, restore request_path to default, and change
* thresholds from defaults */
const testAccComputeHttpHealthCheck_update2 = `
resource "google_compute_http_health_check" "foobar" {
	name = "terraform-test"
	description = "Resource updated for Terraform acceptance testing"
	healthy_threshold = 10
	unhealthy_threshold = 10
}
`
