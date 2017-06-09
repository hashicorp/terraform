package google

import (
	"testing"

	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeHttpHealthCheck_importBasic(t *testing.T) {
	resourceName := "google_compute_http_health_check.foobar"

	hhckName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeHttpHealthCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeHttpHealthCheck_basic(hhckName),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
