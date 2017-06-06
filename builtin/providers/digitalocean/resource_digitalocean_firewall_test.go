package digitalocean

import (
	"context"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanFirewall_Basic(t *testing.T) {
	tests := []struct {
		description  string
		firewallName string
		config       string
		checkers     []resource.TestCheckFunc
	}{
		{
			description:  "only allow inbound SSH(TCP/22)",
			firewallName: fmt.Sprintf("foobar-test-terraform-firewall-%s", acctest.RandString(10)),
			config: `
			resource "digitalocean_firewall" "foobar" {
				name          = "%s"
				inbound_rules = [
				{
					protocol         = "tcp"
					port_range       = "22"
					source_addresses = ["0.0.0.0/0", "::/0"]
				},
				]
			}`,
			checkers: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.#", "1"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.port_range", "22"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.0", "0.0.0.0/0"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.1", "::/0"),
			},
		},
		{
			description:  "only allow outbound SSH(TCP/22)",
			firewallName: fmt.Sprintf("foobar-test-terraform-firewall-%s", acctest.RandString(10)),
			config: `
			resource "digitalocean_firewall" "foobar" {
				name          = "%s"
				outbound_rules = [
				{
					protocol              = "tcp"
					port_range            = "22"
					destination_addresses = ["0.0.0.0/0", "::/0"]
				},
				]
			}`,
			checkers: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.#", "1"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.port_range", "22"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.0", "0.0.0.0/0"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.1", "::/0"),
			},
		},
		{
			description:  "only allow inbound SSH(TCP/22) and HTTP(TCP/80)",
			firewallName: fmt.Sprintf("foobar-test-terraform-firewall-%s", acctest.RandString(10)),
			config: `
			resource "digitalocean_firewall" "foobar" {
				name          = "%s"
				inbound_rules = [
				{
					protocol         = "tcp"
					port_range       = "22"
					source_addresses = ["0.0.0.0/0", "::/0"]
				},
				{
					protocol         = "tcp"
					port_range       = "80"
					source_addresses = ["1.2.3.0/24", "2002::/16"]
				},
				]
			}`,
			checkers: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.port_range", "22"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.0", "0.0.0.0/0"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.1", "::/0"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.port_range", "80"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.source_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.source_addresses.0", "1.2.3.0/24"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.source_addresses.1", "2002::/16"),
			},
		},
		{
			description:  "only allow outbound SSH(TCP/22) and DNS(UDP/53)",
			firewallName: fmt.Sprintf("foobar-test-terraform-firewall-%s", acctest.RandString(10)),
			config: `
			resource "digitalocean_firewall" "foobar" {
				name          = "%s"
				outbound_rules = [
				{
					protocol              = "tcp"
					port_range            = "22"
					destination_addresses = ["192.168.1.0/24", "2002:1001::/48"]
				},
				{
					protocol              = "udp"
					port_range            = "53"
					destination_addresses = ["1.2.3.0/24", "2002::/16"]
				},
				]
			}`,
			checkers: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.port_range", "22"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.0", "192.168.1.0/24"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.1", "2002:1001::/48"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.port_range", "53"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.protocol", "udp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.destination_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.destination_addresses.0", "1.2.3.0/24"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.destination_addresses.1", "2002::/16"),
			},
		},
		{
			description:  "allow inbound and outbound HTTPS(TCP/443), inbound SSH(TCP/22), and outbound DNS(UDP/53)",
			firewallName: fmt.Sprintf("foobar-test-terraform-firewall-%s", acctest.RandString(10)),
			config: `
			resource "digitalocean_firewall" "foobar" {
				name          = "%s"
				inbound_rules = [
				{
					protocol         = "tcp"
					port_range       = "443"
					source_addresses = ["192.168.1.0/24", "2002:1001:1:2::/64"]
				},
				{
					protocol         = "tcp"
					port_range       = "22"
					source_addresses = ["0.0.0.0/0", "::/0"]
				},
				]
				outbound_rules = [
				{
					protocol              = "tcp"
					port_range            = "443"
					destination_addresses = ["192.168.1.0/24", "2002:1001:1:2::/64"]
				},
				{
					protocol              = "udp"
					port_range            = "53"
					destination_addresses = ["0.0.0.0/0", "::/0"]
				},
				]
			}`,
			checkers: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.port_range", "443"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.0", "192.168.1.0/24"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.0.source_addresses.1", "2002:1001:1:2::/64"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.port_range", "22"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.source_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.source_addresses.0", "0.0.0.0/0"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "inbound_rules.1.source_addresses.1", "::/0"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.port_range", "443"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.protocol", "tcp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.0", "192.168.1.0/24"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.0.destination_addresses.1", "2002:1001:1:2::/64"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.port_range", "53"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.protocol", "udp"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.destination_addresses.#", "2"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.destination_addresses.0", "0.0.0.0/0"),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "outbound_rules.1.destination_addresses.1", "::/0"),
			},
		},
	}

	var firewall godo.Firewall
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			checkers := []resource.TestCheckFunc{
				testAccCheckDigitalOceanFirewallExists("digitalocean_firewall.foobar", &firewall),
				resource.TestCheckResourceAttr("digitalocean_firewall.foobar", "name", tt.firewallName),
			}
			checkers = append(checkers, tt.checkers...)
			resource.Test(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				Providers:    testAccProviders,
				CheckDestroy: testAccCheckDigitalOceanFirewallDestroy,
				Steps: []resource.TestStep{
					{
						Config: fmt.Sprintf(tt.config, tt.firewallName),
						Check:  resource.ComposeTestCheckFunc(checkers...),
					},
				},
			})
		})
	}
}

func testAccCheckDigitalOceanFirewallDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_firewall" {
			continue
		}

		// Try to find the firewall
		_, _, err := client.Firewalls.Get(context.Background(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Firewall still exists")
		}
	}

	return nil
}

func testAccCheckDigitalOceanFirewallExists(n string, firewall *godo.Firewall) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		foundFirewall, _, err := client.Firewalls.Get(context.Background(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundFirewall.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*firewall = *foundFirewall

		return nil
	}
}
