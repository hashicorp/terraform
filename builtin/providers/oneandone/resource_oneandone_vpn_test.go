package oneandone

import (
	"fmt"
	"testing"

	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"time"
)

func TestAccOneandoneVpn_Basic(t *testing.T) {
	var server oneandone.VPN

	name := "test"
	name_updated := "test1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDOneandoneVPNDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneVPN_basic, name),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneVPNExists("oneandone_vpn.vpn", &server),
					testAccCheckOneandoneVPNAttributes("oneandone_vpn.vpn", name),
					resource.TestCheckResourceAttr("oneandone_vpn.vpn", "name", name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneVPN_basic, name_updated),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneVPNExists("oneandone_vpn.vpn", &server),
					testAccCheckOneandoneVPNAttributes("oneandone_vpn.vpn", name_updated),
					resource.TestCheckResourceAttr("oneandone_vpn.vpn", "name", name_updated),
				),
			},
		},
	})
}

func testAccCheckDOneandoneVPNDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_server" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetVPN(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("VPN still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandoneVPNAttributes(n string, reverse_dns string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != reverse_dns {
			return fmt.Errorf("Bad name: expected %s : found %s ", reverse_dns, rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckOneandoneVPNExists(n string, server *oneandone.VPN) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_server, err := api.GetVPN(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching VPN: %s", rs.Primary.ID)
		}
		if found_server.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		server = found_server

		return nil
	}
}

const testAccCheckOneandoneVPN_basic = `
resource "oneandone_vpn" "vpn" {
  datacenter = "GB"
  name = "%s"
  description = "ttest descr"
}`
