package opc

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccOPCIPAddressReservation_Basic(t *testing.T) {
	rInt := acctest.RandInt()
	resName := "opc_compute_ip_address_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOPCCheckIPAddressReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOPCIPAddressReservationConfig_Basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccOPCCheckIPAddressReservationExists,
					resource.TestCheckResourceAttr(resName, "name", fmt.Sprintf("testing-ip-address-reservation-%d", rInt)),
				),
			},
		},
	})
}

func testAccOPCCheckIPAddressReservationExists(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAddressReservations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_address_reservation" {
			continue
		}

		input := compute.GetIPAddressReservationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if _, err := client.GetIPAddressReservation(&input); err != nil {
			return fmt.Errorf("Error retrieving state of IP Address Reservation %s: %s", input.Name, err)
		}
	}

	return nil
}
func testAccOPCCheckIPAddressReservationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*compute.Client).IPAddressReservations()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opc_compute_ip_address_reservation" {
			continue
		}

		input := compute.GetIPAddressReservationInput{
			Name: rs.Primary.Attributes["name"],
		}
		if info, err := client.GetIPAddressReservation(&input); err == nil {
			return fmt.Errorf("IP Address Reservation %s still exists: %#v", input.Name, info)
		}
	}

	return nil
}

func testAccOPCIPAddressReservationConfig_Basic(rInt int) string {
	return fmt.Sprintf(`
resource "opc_compute_ip_address_reservation" "test" {
  name = "testing-ip-address-reservation-%d"
  description = "testing-desc-%d"
  ip_address_pool = "public-ippool"
}`, rInt, rInt)
}
