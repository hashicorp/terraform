package scaleway

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccScalewayVolumeAttachment_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckScalewayVolumeAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckScalewayVolumeAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVolumeAttachmentExists("scaleway_volume_attachment.test"),
				),
			},
		},
	})
}

func testAccCheckScalewayVolumeAttachmentDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).scaleway

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "scaleway" {
			continue
		}

		s, err := client.GetServer(rs.Primary.Attributes["server"])
		if err != nil {
			fmt.Printf("Failed getting server: %q", err)
			return err
		}

		for _, volume := range s.Volumes {
			if volume.Identifier == rs.Primary.Attributes["volume"] {
				return fmt.Errorf("Attachment still exists")
			}
		}
	}

	return nil
}

func testAccCheckScalewayVolumeAttachmentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client).scaleway

		rs, _ := s.RootModule().Resources[n]

		server, err := client.GetServer(rs.Primary.Attributes["server"])
		if err != nil {
			fmt.Printf("Failed getting server: %q", err)
			return err
		}

		for _, volume := range server.Volumes {
			if volume.Identifier == rs.Primary.Attributes["volume"] {
				return nil
			}
		}

		return fmt.Errorf("Attachment does not exist")
	}
}

var testAccCheckScalewayVolumeAttachmentConfig = fmt.Sprintf(`
resource "scaleway_server" "base" {
  name = "test"
  # ubuntu 14.04
  image = "%s"
  type = "C1"
  # state = "stopped"
}

resource "scaleway_volume" "test" {
  name = "test"
  size_in_gb = 5
  type = "l_ssd"
}

resource "scaleway_volume_attachment" "test" {
  server = "${scaleway_server.base.id}"
  volume = "${scaleway_volume.test.id}"
}`, armImageIdentifier)
