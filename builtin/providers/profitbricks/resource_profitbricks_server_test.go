package profitbricks

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func TestAccProfitBricksServer_Basic(t *testing.T) {
	var server profitbricks.Server
	serverName := "webserver"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDProfitBricksServerDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckProfitbricksServerConfig_basic, serverName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksServerExists("profitbricks_server.webserver", &server),
					testAccCheckProfitBricksServerAttributes("profitbricks_server.webserver", serverName),
					resource.TestCheckResourceAttr("profitbricks_server.webserver", "name", serverName),
				),
			},
			resource.TestStep{
				Config: testAccCheckProfitbricksServerConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksServerAttributes("profitbricks_server.webserver", "updated"),
					resource.TestCheckResourceAttr("profitbricks_server.webserver", "name", "updated"),
				),
			},
		},
	})
}

func testAccCheckDProfitBricksServerDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "profitbricks_datacenter" {
			continue
		}

		resp := profitbricks.GetServer(rs.Primary.Attributes["datacenter_id"], rs.Primary.ID)

		if resp.StatusCode < 299 {
			return fmt.Errorf("Server still exists %s %s", rs.Primary.ID, resp.Response)
		}
	}

	return nil
}

func testAccCheckProfitBricksServerAttributes(n string, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksServerAttributes: Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("Bad name: %s", rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckProfitBricksServerExists(n string, server *profitbricks.Server) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksServerExists: Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		foundServer := profitbricks.GetServer(rs.Primary.Attributes["datacenter_id"], rs.Primary.ID)

		if foundServer.StatusCode != 200 {
			return fmt.Errorf("Error occured while fetching Server: %s", rs.Primary.ID)
		}
		if foundServer.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		server = &foundServer

		return nil
	}
}

const testAccCheckProfitbricksServerConfig_basic = `
resource "profitbricks_datacenter" "foobar" {
	name       = "server-test"
	location = "us/las"
}

resource "profitbricks_lan" "webserver_lan" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  public = true
  name = "public"
}

resource "profitbricks_server" "webserver" {
  name = "%s"
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
    lan = "${profitbricks_lan.webserver_lan.id}"
    dhcp = true
    firewall_active = true
    firewall {
      protocol = "TCP"
      name = "SSH"
      port_range_start = 22
      port_range_end = 22
    }
  }
}`

const testAccCheckProfitbricksServerConfig_update = `
resource "profitbricks_datacenter" "foobar" {
	name       = "server-test"
	location = "us/las"
}

resource "profitbricks_server" "webserver" {
  name = "updated"
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
}`
