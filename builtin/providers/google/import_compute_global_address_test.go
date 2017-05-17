package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeGlobalAddress_importBasic(t *testing.T) {
	resourceName := "google_compute_global_address.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeGlobalAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeGlobalAddress_basic,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
