package digitalocean

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDigitalOceanFirewall_importBasic(t *testing.T) {
	tests := []struct {
		description  string
		firewallName string
		config       string
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			config := fmt.Sprintf(tt.config, tt.firewallName)
			resourceName := fmt.Sprintf("digitalocean_firewall.%s", tt.firewallName)
			resource.Test(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				Providers:    testAccProviders,
				CheckDestroy: testAccCheckDigitalOceanFirewallDestroy,
				Steps: []resource.TestStep{
					{
						Config:            config,
						ResourceName:      resourceName,
						ImportState:       true,
						ImportStateVerify: true,
					},
				},
			})
		})
	}
}
