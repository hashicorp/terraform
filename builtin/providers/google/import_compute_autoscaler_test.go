package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAutoscaler_importBasic(t *testing.T) {
	resourceName := "google_compute_autoscaler.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAutoscalerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAutoscaler_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
