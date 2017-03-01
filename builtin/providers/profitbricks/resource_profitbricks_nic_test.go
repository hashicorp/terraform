package profitbricks

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func TestAccProfitBricksNic_Basic(t *testing.T) {
	var nic profitbricks.Nic
	volumeName := "volume"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDProfitBricksNicDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckProfitbricksNicConfig_basic, volumeName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksNICExists("profitbricks_nic.database_nic", &nic),
					testAccCheckProfitBricksNicAttributes("profitbricks_nic.database_nic", volumeName),
					resource.TestCheckResourceAttr("profitbricks_nic.database_nic", "name", volumeName),
				),
			},
			resource.TestStep{
				Config: testAccCheckProfitbricksNicConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksNicAttributes("profitbricks_nic.database_nic", "updated"),
					resource.TestCheckResourceAttr("profitbricks_nic.database_nic", "name", "updated"),
				),
			},
		},
	})
}

func testAccCheckDProfitBricksNicDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "profitbricks_nic" {
			continue
		}

		resp := profitbricks.GetNic(rs.Primary.Attributes["datacenter_id"], rs.Primary.Attributes["nic_id"], rs.Primary.ID)

		if resp.StatusCode < 299 {
			return fmt.Errorf("NIC still exists %s %s", rs.Primary.ID, resp.Response)
		}
	}

	return nil
}

func testAccCheckProfitBricksNicAttributes(n string, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksNicAttributes: Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("Bad name: %s", rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckProfitBricksNICExists(n string, nic *profitbricks.Nic) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksVolumeExists: Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		foundNic := profitbricks.GetNic(rs.Primary.Attributes["datacenter_id"], rs.Primary.Attributes["server_id"], rs.Primary.ID)

		if foundNic.StatusCode != 200 {
			return fmt.Errorf("Error occured while fetching Volume: %s", rs.Primary.ID)
		}
		if foundNic.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		nic = &foundNic

		return nil
	}
}

const testAccCheckProfitbricksNicConfig_basic = `
resource "profitbricks_datacenter" "foobar" {
	name       = "nic-test"
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
    image_password = "K3tTj8G14a3EgKyNeeiY"
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
  name = "%s"
}`

const testAccCheckProfitbricksNicConfig_update = `
resource "profitbricks_datacenter" "foobar" {
	name       = "nic-test"
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
    image_password = "K3tTj8G14a3EgKyNeeiY"
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
`
