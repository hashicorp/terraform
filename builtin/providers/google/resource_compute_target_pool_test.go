package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeTargetPool_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetPoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetPool_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetPoolExists(
						"google_compute_target_pool.foobar"),
				),
			},
		},
	})
}

func testAccCheckComputeTargetPoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_target_pool" {
			continue
		}

		_, err := config.clientCompute.TargetPools.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("TargetPool still exists")
		}
	}

	return nil
}

func testAccCheckComputeTargetPoolExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.TargetPools.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("TargetPool not found")
		}

		return nil
	}
}

var testAccComputeTargetPool_basic = fmt.Sprintf(`
resource "google_compute_http_health_check" "foobar" {
	name = "healthcheck-test-%s"
	host = "example.com"
}

resource "google_compute_target_pool" "foobar" {
	description = "Resource created for Terraform acceptance testing"
	instances = ["us-central1-a/foo", "us-central1-b/bar"]
	name = "tpool-test-%s"
	session_affinity = "CLIENT_IP_PROTO"
	health_checks = [
		"${google_compute_http_health_check.foobar.name}"
	]
}`, acctest.RandString(10), acctest.RandString(10))
