package packet

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/packethost/packngo"
)

func TestAccPacketVolume_Basic(t *testing.T) {
	var volume packngo.Volume

	project_id := os.Getenv("PACKET_PROJECT_ID")
	facility := os.Getenv("PACKET_FACILITY")

	resource.Test(t, resource.TestCase{
		PreCheck:     testAccPacketVolumePreCheck(t),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPacketVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckPacketVolumeConfig_basic, project_id, facility),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPacketVolumeExists("packet_volume.foobar", &volume),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "project_id", project_id),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "plan", "storage_1"),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "billing_cycle", "hourly"),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "size", "100"),
				),
			},
		},
	})
}

func testAccCheckPacketVolumeDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*packngo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "packet_volume" {
			continue
		}
		if _, _, err := client.Volumes.Get(rs.Primary.ID); err == nil {
			return fmt.Errorf("Volume still exists")
		}
	}

	return nil
}

func testAccCheckPacketVolumeExists(n string, volume *packngo.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		client := testAccProvider.Meta().(*packngo.Client)

		foundVolume, _, err := client.Volumes.Get(rs.Primary.ID)
		if err != nil {
			return err
		}
		if foundVolume.ID != rs.Primary.ID {
			return fmt.Errorf("Record not found: %v - %v", rs.Primary.ID, foundVolume)
		}

		*volume = *foundVolume

		return nil
	}
}

func testAccPacketVolumePreCheck(t *testing.T) func() {
	return func() {
		testAccPreCheck(t)
		if os.Getenv("PACKET_PROJECT_ID") == "" {
			t.Fatal("PACKET_PROJECT_ID must be set")
		}
		if os.Getenv("PACKET_FACILITY") == "" {
			t.Fatal("PACKET_FACILITY must be set")
		}
	}
}

const testAccCheckPacketVolumeConfig_basic = `
resource "packet_volume" "foobar" {
    plan = "storage_1"
    billing_cycle = "hourly"
    size = 100
    project_id = "%s"
    facility = "%s"
}`
