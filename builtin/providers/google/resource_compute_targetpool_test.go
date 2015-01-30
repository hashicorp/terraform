package google

import (
	"fmt"
	"testing"

	"code.google.com/p/google-api-go-client/compute/v1"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccComputeTargetpoolBasic(t *testing.T) {
	var targetpool compute.TargetPool

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetpoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetpoolBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeTargetpoolExists(
						"google_compute_targetpool.foobar", &targetpool),
				),
			},
		},
	})
}

func testAccCheckComputeTargetpoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_targetpool" {
			continue
		}

		_, err := config.clientCompute.TargetPools.Get(
			config.Project, config.Region, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Targetpool still exists")
		}
	}

	return nil
}

func testAccCheckComputeTargetpoolExists(n string, targetpool *compute.TargetPool) resource.TestCheckFunc {
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
			return fmt.Errorf("Targetpool not found")
		}

		*targetpool = *found

		return nil
	}
}

const testAccComputeTargetpoolBasic = `
resource "google_compute_targetpool" "foobar" {
	name = "terraform-test"
}`
