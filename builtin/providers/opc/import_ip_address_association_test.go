package opc

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCIPAddressAssociation_importBasic(t *testing.T) {
	resourceName := "opc_compute_ip_address_association.test"

	ri := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIPAddressAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIPAddressAssociationBasic(ri),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
