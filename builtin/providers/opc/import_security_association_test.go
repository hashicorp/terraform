package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCSecurityAssociation_importBasic(t *testing.T) {
	resourceName := "opc_compute_security_association.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccSecurityAssociationBasic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityAssociationDestroy,
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

func TestAccOPCSecurityAssociation_importComplete(t *testing.T) {
	resourceName := "opc_compute_security_association.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccSecurityAssociationComplete, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckSecurityAssociationDestroy,
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
