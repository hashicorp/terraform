package google

import (
	"testing"

	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeAutoscaler_importBasic(t *testing.T) {
	resourceName := "google_compute_autoscaler.foobar"

	var it_name = fmt.Sprintf("autoscaler-test-%s", acctest.RandString(10))
	var tp_name = fmt.Sprintf("autoscaler-test-%s", acctest.RandString(10))
	var igm_name = fmt.Sprintf("autoscaler-test-%s", acctest.RandString(10))
	var autoscaler_name = fmt.Sprintf("autoscaler-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeAutoscalerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeAutoscaler_basic(it_name, tp_name, igm_name, autoscaler_name),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
