package packet

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/packethost/packngo"
)

func TestAccPacketVolume_Basic(t *testing.T) {
	var volume packngo.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPacketVolumeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckPacketVolumeConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPacketVolumeExists("packet_volume.foobar", &volume),
					testAccCheckPacketVolumeAttributes(&volume),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "project_id", "foobar"),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "plan", "foobar"),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "facility", "foobar"),
					resource.TestCheckResourceAttr(
						"packet_volume.foobar", "billing_cycle", "hourly"),
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
			return fmt.Errorf("Volume cstill exists")
		}
	}

	return nil
}

func testAccCheckPacketVolumeAttributes(volume *packngo.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if volume.Name != "foobar" {
			return fmt.Errorf("Bad name: %s", volume.Name)
		}
		return nil
	}
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

var testAccCheckPacketVolumeConfig_basic = fmt.Sprintf(`
resource "packet_volume" "foobar" {
    project_id = "foobar"
    plan = "foobar"
    facility = "foobar"
    billing_cycle = "hourly"
}`)
