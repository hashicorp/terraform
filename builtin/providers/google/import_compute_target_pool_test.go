package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeTargetPool_importBasic(t *testing.T) {
	resourceName := "google_compute_target_pool.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeTargetPoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeTargetPool_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
