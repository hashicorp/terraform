package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCInstance_importBasic(t *testing.T) {
	rInt := acctest.RandInt()

	resourceName := "opc_compute_instance.test"
	instanceName := fmt.Sprintf("acc-test-instance-%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceBasic(rInt),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdPrefix:     instanceName + "/",
				ImportStateVerifyIgnore: []string{"instance_attributes"},
			},
		},
	})
}
