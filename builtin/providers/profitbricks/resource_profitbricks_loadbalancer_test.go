package profitbricks

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func TestAccProfitBricksLoadbalancer_Basic(t *testing.T) {
	var loadbalancer profitbricks.Loadbalancer
	lbName := "loadbalancer"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDProfitBricksLoadbalancerDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckProfitbricksLoadbalancerConfig_basic, lbName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksLoadbalancerExists("profitbricks_loadbalancer.example", &loadbalancer),
					testAccCheckProfitBricksLoadbalancerAttributes("profitbricks_loadbalancer.example", lbName),
					resource.TestCheckResourceAttr("profitbricks_loadbalancer.example", "name", lbName),
				),
			},
			resource.TestStep{
				Config: testAccCheckProfitbricksLoadbalancerConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksLoadbalancerAttributes("profitbricks_loadbalancer.example", "updated"),
					resource.TestCheckResourceAttr("profitbricks_loadbalancer.example", "name", "updated"),
				),
			},
		},
	})
}

func testAccCheckDProfitBricksLoadbalancerDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "profitbricks_loadbalancer" {
			continue
		}

		resp := profitbricks.GetLoadbalancer(rs.Primary.Attributes["datacenter_id"], rs.Primary.ID)

		if resp.StatusCode < 299 {
			return fmt.Errorf("Firewall still exists %s %s", rs.Primary.ID, resp.Response)
		}
	}

	return nil
}

func testAccCheckProfitBricksLoadbalancerAttributes(n string, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksLoadbalancerAttributes: Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("Bad name: %s", rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckProfitBricksLoadbalancerExists(n string, loadbalancer *profitbricks.Loadbalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksLoadbalancerExists: Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		foundLB := profitbricks.GetLoadbalancer(rs.Primary.Attributes["datacenter_id"], rs.Primary.ID)

		if foundLB.StatusCode != 200 {
			return fmt.Errorf("Error occured while fetching Loadbalancer: %s", rs.Primary.ID)
		}
		if foundLB.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		loadbalancer = &foundLB

		return nil
	}
}

const testAccCheckProfitbricksLoadbalancerConfig_basic = `
resource "profitbricks_datacenter" "foobar" {
	name       = "loadbalancer-test"
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
  lan = "2"
  dhcp = true
  firewall_active = true
  name = "updated"
}

resource "profitbricks_loadbalancer" "example" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  nic_id = "${profitbricks_nic.database_nic.id}"
  name = "%s"
  dhcp = true
}`

const testAccCheckProfitbricksLoadbalancerConfig_update = `
resource "profitbricks_datacenter" "foobar" {
	name       = "loadbalancer-test"
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
  lan = "2"
  dhcp = true
  firewall_active = true
  name = "updated"
}

resource "profitbricks_loadbalancer" "example" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  nic_id = "${profitbricks_nic.database_nic.id}"
  name = "updated"
  dhcp = true
}`
