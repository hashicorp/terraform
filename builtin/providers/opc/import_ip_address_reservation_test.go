package opc

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccOPCIPAddressReservation_importBasic(t *testing.T) {
	resourceName := "opc_compute_ip_address_reservation.test"

	ri := acctest.RandInt()
	config := testAccOPCIPAddressReservationConfig_Basic(ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckIPAddressReservationDestroy,
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
func TestAccOPCIPAddressReservation_importDisabled(t *testing.T) {
	resourceName := "opc_compute_ip_address_reservation.test"

	ri := acctest.RandInt()
	config := testAccOPCIPAddressReservationConfig_Basic(ri)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckIPAddressReservationDestroy,
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
