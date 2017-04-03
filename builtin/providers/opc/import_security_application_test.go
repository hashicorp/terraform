package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCSecurityApplication_importICMP(t *testing.T) {
	resourceName := "opc_compute_security_application.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityApplicationICMP, ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityApplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOPCSecurityApplication_importTCP(t *testing.T) {
	resourceName := "opc_compute_security_application.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccOPCSecurityApplicationTCP, ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityApplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
