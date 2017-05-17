package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var testAccVRackPublicCloudAttachmentConfig = fmt.Sprintf(`
resource "ovh_vrack_publiccloud_attachment" "attach" {
  vrack_id = "%s"
  project_id = "%s"
}
`, os.Getenv("OVH_VRACK"), os.Getenv("OVH_PUBLIC_CLOUD"))

func TestAccVRackPublicCloudAttachment_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCheckVRackPublicCloudAttachmentPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVRackPublicCloudAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVRackPublicCloudAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVRackPublicCloudAttachmentExists("ovh_vrack_publiccloud_attachment.attach", t),
				),
			},
		},
	})
}

func testAccCheckVRackPublicCloudAttachmentPreCheck(t *testing.T) {
	testAccPreCheck(t)
	testAccCheckVRackExists(t)
	testAccCheckPublicCloudExists(t)
}

func testAccCheckVRackPublicCloudAttachmentExists(n string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.Attributes["vrack_id"] == "" {
			return fmt.Errorf("No VRack ID is set")
		}

		if rs.Primary.Attributes["project_id"] == "" {
			return fmt.Errorf("No Project ID is set")
		}

		return vrackPublicCloudAttachmentExists(rs.Primary.Attributes["vrack_id"], rs.Primary.Attributes["project_id"], config.OVHClient)
	}
}

func testAccCheckVRackPublicCloudAttachmentDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ovh_vrack_publiccloud_attachment" {
			continue
		}

		err := vrackPublicCloudAttachmentExists(rs.Primary.Attributes["vrack_id"], rs.Primary.Attributes["project_id"], config.OVHClient)
		if err == nil {
			return fmt.Errorf("VRack > Public Cloud Attachment still exists")
		}

	}
	return nil
}
