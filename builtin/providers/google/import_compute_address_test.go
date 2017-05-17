package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeAddress_importBasic(t *testing.T) {
	resourceName := "google_compute_address.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeAddress_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
