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

func TestAccOneandoneFirewall_Basic(t *testing.T) {
	var firewall oneandone.FirewallPolicy

	name := "test"
	name_updated := "test1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDOneandoneFirewallDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneFirewall_basic, name),

				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneFirewallExists("oneandone_firewall_policy.fw", &firewall),
					testAccCheckOneandoneFirewallAttributes("oneandone_firewall_policy.fw", name),
					resource.TestCheckResourceAttr("oneandone_firewall_policy.fw", "name", name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneFirewall_update, name_updated),

				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneFirewallExists("oneandone_firewall_policy.fw", &firewall),
					testAccCheckOneandoneFirewallAttributes("oneandone_firewall_policy.fw", name_updated),
					resource.TestCheckResourceAttr("oneandone_firewall_policy.fw", "name", name_updated),
				),
			},
		},
	})
}

func testAccCheckDOneandoneFirewallDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_firewall_policy.fw" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetFirewallPolicy(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Firewall Policy still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandoneFirewallAttributes(n string, reverse_dns string) resource.TestCheckFunc {
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

func testAccCheckOneandoneFirewallExists(n string, fw_p *oneandone.FirewallPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_fw, err := api.GetFirewallPolicy(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching Firewall Policy: %s", rs.Primary.ID)
		}
		if found_fw.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		fw_p = found_fw

		return nil
	}
}

const testAccCheckOneandoneFirewall_basic = `
resource "oneandone_firewall_policy" "fw" {
  name = "%s"
  rules = [
    {
      "protocol" = "TCP"
      "port_from" = 80
      "port_to" = 80
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "ICMP"
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "TCP"
      "port_from" = 43
      "port_to" = 43
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "TCP"
      "port_from" = 22
      "port_to" = 22
      "source_ip" = "0.0.0.0"
    }
  ]
}`

const testAccCheckOneandoneFirewall_update = `
resource "oneandone_firewall_policy" "fw" {
  name = "%s"
  rules = [
    {
      "protocol" = "TCP"
      "port_from" = 80
      "port_to" = 80
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "ICMP"
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "TCP"
      "port_from" = 43
      "port_to" = 43
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "TCP"
      "port_from" = 22
      "port_to" = 22
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "TCP"
      "port_from" = 88
      "port_to" = 88
      "source_ip" = "0.0.0.0"
    },
  ]
}`
