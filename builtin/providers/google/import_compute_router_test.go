package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeRouter_import(t *testing.T) {
	resourceName := "google_compute_router.foobar"
	network := fmt.Sprintf("router-import-test-%s", acctest.RandString(10))
	subnet := fmt.Sprintf("router-import-test-%s", acctest.RandString(10))
	router := fmt.Sprintf("router-import-test-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeRouterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouterNetworkLink(network, subnet, router),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
