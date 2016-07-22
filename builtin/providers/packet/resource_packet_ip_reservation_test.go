package packet

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/packethost/packngo"
)

func TestAccPacketIPReservation_Basic(t *testing.T) {
	var ip_reservation packngo.IPReservation

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPacketIPReservationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPacketIPReservationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPacketIPReservationExists("packet_ip_reservation.foobar", &ip_reservation),
					testAccCheckPacketIPReservationAttributes(&ip_reservation),
					resource.TestCheckResourceAttr(
						"packet_ip_reservation.foobar", "address", "foobar"),
				),
			},
		},
	})
}

func testAccCheckPacketIPReservationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*packngo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "packet_ip_reservation" {
			continue
		}
		if _, _, err := client.IpReservations.Get(rs.Primary.ID); err == nil {
			return fmt.Errorf("IPReservation cstill exists")
		}
	}

	return nil
}

func testAccCheckPacketIPReservationAttributes(ip_reservation *packngo.IPReservation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if ip_reservation.Address != "foobar" {
			return fmt.Errorf("Bad address: %s", ip_reservation.Address)
		}
		return nil
	}
}

func testAccCheckPacketIPReservationExists(n string, ip_reservation *packngo.IPReservation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*packngo.Client)

		foundIPReservation, _, err := client.IpReservations.Get(rs.Primary.ID)
		if err != nil {
			return err
		}
		if foundIPReservation.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found: %v - %v", rs.Primary.ID, foundIPReservation)
		}

		*ip_reservation = *foundIPReservation

		return nil
	}
}

var testAccCheckPacketIPReservationConfig_basic = fmt.Sprintf(`
resource "packet_ip_reservation" "foobar" {
    address = "foobar"
}`)
