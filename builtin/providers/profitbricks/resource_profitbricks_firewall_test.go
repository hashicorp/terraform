package profitbricks

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func TestAccProfitBricksFirewall_Basic(t *testing.T) {
	var firewall profitbricks.FirewallRule
	firewallName := "firewall"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDProfitBricksFirewallDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckProfitbricksFirewallConfig_basic, firewallName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksFirewallExists("profitbricks_firewall.webserver_http", &firewall),
					testAccCheckProfitBricksFirewallAttributes("profitbricks_firewall.webserver_http", firewallName),
					resource.TestCheckResourceAttr("profitbricks_firewall.webserver_http", "name", firewallName),
				),
			},
			resource.TestStep{
				Config: testAccCheckProfitbricksFirewallConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksFirewallAttributes("profitbricks_firewall.webserver_http", "updated"),
					resource.TestCheckResourceAttr("profitbricks_firewall.webserver_http", "name", "updated"),
				),
			},
		},
	})
}

func testAccCheckDProfitBricksFirewallDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "profitbricks_firewall" {
			continue
		}

		resp := profitbricks.GetFirewallRule(rs.Primary.Attributes["datacenter_id"], rs.Primary.Attributes["server_id"], rs.Primary.Attributes["nic_id"], rs.Primary.ID)

		if resp.StatusCode < 299 {
			return fmt.Errorf("Firewall still exists %s %s", rs.Primary.ID, resp.Response)
		}
	}

	return nil
}

func testAccCheckProfitBricksFirewallAttributes(n string, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksFirewallAttributes: Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("Bad name: %s", rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckProfitBricksFirewallExists(n string, firewall *profitbricks.FirewallRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksFirewallExists: Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		foundServer := profitbricks.GetFirewallRule(rs.Primary.Attributes["datacenter_id"], rs.Primary.Attributes["server_id"], rs.Primary.Attributes["nic_id"], rs.Primary.ID)

		if foundServer.StatusCode != 200 {
			return fmt.Errorf("Error occured while fetching Firewall rule: %s", rs.Primary.ID)
		}
		if foundServer.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		firewall = &foundServer

		return nil
	}
}

const testAccCheckProfitbricksFirewallConfig_basic = `
resource "profitbricks_datacenter" "foobar" {
	name       = "firewall-test"
	location = "us/las"
}

resource "profitbricks_server" "webserver" {
  name = "webserver"
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  cores = 1
  ram = 1024
  availability_zone = "ZONE_1"
  cpu_family = "AMD_OPTERON"
  volume {
    name = "system"
    size = 5
    disk_type = "SSD"
    image_name ="ubuntu-16.04"
    image_password = "test1234"
}
  nic {
    lan = "1"
    dhcp = true
    firewall_active = true
    firewall {
      protocol = "TCP"
      name = "SSH"
      port_range_start = 22
      port_range_end = 22
    }
  }
}

resource "profitbricks_nic" "database_nic" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  server_id = "${profitbricks_server.webserver.id}"
  lan = 2
  dhcp = true
  firewall_active = true
  name = "updated"
}

resource "profitbricks_firewall" "webserver_http" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  server_id = "${profitbricks_server.webserver.id}"
  nic_id = "${profitbricks_nic.database_nic.id}"
  protocol = "TCP"
  name = "%s"
  port_range_start = 80
  port_range_end = 80
}`

const testAccCheckProfitbricksFirewallConfig_update = `
resource "profitbricks_datacenter" "foobar" {
	name       = "firewall-test"
	location = "us/las"
}

resource "profitbricks_server" "webserver" {
  name = "webserver"
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  cores = 1
  ram = 1024
  availability_zone = "ZONE_1"
  cpu_family = "AMD_OPTERON"
  volume {
    name = "system"
    size = 5
    disk_type = "SSD"
    image_name ="ubuntu-16.04"
    image_password = "test1234"
}
  nic {
    lan = "1"
    dhcp = true
    firewall_active = true
    firewall {
      protocol = "TCP"
      name = "SSH"
      port_range_start = 22
      port_range_end = 22
    }
  }
}

resource "profitbricks_nic" "database_nic" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  server_id = "${profitbricks_server.webserver.id}"
  lan = 2
  dhcp = true
  firewall_active = true
  name = "updated"
}

resource "profitbricks_firewall" "webserver_http" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  server_id = "${profitbricks_server.webserver.id}"
  nic_id = "${profitbricks_nic.database_nic.id}"
  protocol = "TCP"
  name = "updated"
  port_range_start = 80
  port_range_end = 80
}`
