package icanhazip

import (
	"fmt"
	"net"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccIcanhazipIPAddress_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIcanhazipIPAddressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccIcanhazipIPAddress("data.icanhazip_ipaddress.localip"),
				),
			},
		},
	})
}

func testAccIcanhazipIPAddress(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		r := s.RootModule().Resources[n]
		a := r.Primary.Attributes

		ipaddress := a["ip_address"]

		if ip := net.ParseIP(ipaddress); ip == nil {
			return fmt.Errorf("Not a valid IP address: %s", ipaddress)
		}

		return nil
	}
}

const testAccIcanhazipIPAddressConfig = `
data "icanhazip_ipaddress" "localip" { }
`
