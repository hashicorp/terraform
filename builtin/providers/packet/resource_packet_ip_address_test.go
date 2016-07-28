package packet

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/packethost/packngo"
)

func TestAccPacketIPAddress_Basic(t *testing.T) {
	var ip_address packngo.IPAddress

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPacketIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPacketIPAddressConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPacketIPAddressExists("packet_ip_address.foobar", &ip_address),
					testAccCheckPacketIPAddressAttributes(&ip_address),
					resource.TestCheckResourceAttr(
						"packet_ip_address.foobar", "address", "foobar"),
				),
			},
		},
	})
}

func testAccCheckPacketIPAddressDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*packngo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "packet_ip_address" {
			continue
		}
		if _, _, err := client.Ips.Get(rs.Primary.ID); err == nil {
			return fmt.Errorf("IPAddress cstill exists")
		}
	}

	return nil
}

func testAccCheckPacketIPAddressAttributes(ip_address *packngo.IPAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if ip_address.Address != "foobar" {
			return fmt.Errorf("Bad address: %s", ip_address.Address)
		}
		return nil
	}
}

func testAccCheckPacketIPAddressExists(n string, ip_address *packngo.IPAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*packngo.Client)

		foundIPAddress, _, err := client.Ips.Get(rs.Primary.ID)
		if err != nil {
			return err
		}
		if foundIPAddress.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found: %v - %v", rs.Primary.ID, foundIPAddress)
		}

		*ip_address = *foundIPAddress

		return nil
	}
}

var testAccCheckPacketIPAddressConfig_basic = fmt.Sprintf(`
resource "packet_ip_address" "foobar" {
    address = "foobar"
}`)
