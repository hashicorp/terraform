package profitbricks

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func TestAccProfitBricksVolume_Basic(t *testing.T) {
	var volume profitbricks.Volume
	volumeName := "volume"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDProfitBricksVolumeDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckProfitbricksVolumeConfig_basic, volumeName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksVolumeExists("profitbricks_volume.database_volume", &volume),
					testAccCheckProfitBricksVolumeAttributes("profitbricks_volume.database_volume", volumeName),
					resource.TestCheckResourceAttr("profitbricks_volume.database_volume", "name", volumeName),
				),
			},
			resource.TestStep{
				Config: testAccCheckProfitbricksVolumeConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProfitBricksVolumeAttributes("profitbricks_volume.database_volume", "updated"),
					resource.TestCheckResourceAttr("profitbricks_volume.database_volume", "name", "updated"),
				),
			},
		},
	})
}

func testAccCheckDProfitBricksVolumeDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "profitbricks_datacenter" {
			continue
		}

		resp := profitbricks.GetVolume(rs.Primary.Attributes["datacenter_id"], rs.Primary.ID)

		if resp.StatusCode < 299 {
			return fmt.Errorf("Volume still exists %s %s", rs.Primary.ID, resp.Response)
		}
	}

	return nil
}

func testAccCheckProfitBricksVolumeAttributes(n string, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksVolumeAttributes: Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("Bad name: %s", rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckProfitBricksVolumeExists(n string, volume *profitbricks.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("testAccCheckProfitBricksVolumeExists: Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		foundServer := profitbricks.GetVolume(rs.Primary.Attributes["datacenter_id"], rs.Primary.ID)

		if foundServer.StatusCode != 200 {
			return fmt.Errorf("Error occured while fetching Volume: %s", rs.Primary.ID)
		}
		if foundServer.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		volume = &foundServer

		return nil
	}
}

const testAccCheckProfitbricksVolumeConfig_basic = `
resource "profitbricks_datacenter" "foobar" {
	name       = "volume-test"
	location = "us/las"
}

resource "profitbricks_lan" "webserver_lan" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  public = true
  name = "public"
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
}

resource "profitbricks_volume" "database_volume" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  server_id = "${profitbricks_server.webserver.id}"
  licence_type = "OTHER"
  name = "%s"
  size = 5
  disk_type = "SSD"
  bus = "VIRTIO"
}`

const testAccCheckProfitbricksVolumeConfig_update = `
resource "profitbricks_datacenter" "foobar" {
	name       = "volume-test"
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

resource "profitbricks_volume" "database_volume" {
  datacenter_id = "${profitbricks_datacenter.foobar.id}"
  server_id = "${profitbricks_server.webserver.id}"
  licence_type = "OTHER"
  name = "updated"
  size = 5
  disk_type = "SSD"
  bus = "VIRTIO"
}`
