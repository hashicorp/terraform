package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var testAccPublicCloudPrivateNetworkConfig = fmt.Sprintf(`
resource "ovh_vrack_publiccloud_attachment" "attach" {
  vrack_id = "%s"
  project_id = "%s"
}

data "ovh_publiccloud_regions" "regions" {
  project_id = "${ovh_vrack_publiccloud_attachment.attach.project_id}"
}

data "ovh_publiccloud_region" "region_attr" {
  count = 2
  project_id = "${data.ovh_publiccloud_regions.regions.project_id}"
  name = "${element(data.ovh_publiccloud_regions.regions.names, count.index)}"
}

resource "ovh_publiccloud_private_network" "network" {
  project_id  = "${ovh_vrack_publiccloud_attachment.attach.project_id}"
  vlan_id = 0
  name = "terraform_testacc_private_net"
  regions     = ["${data.ovh_publiccloud_regions.regions.names}"]
}
`, os.Getenv("OVH_VRACK"), os.Getenv("OVH_PUBLIC_CLOUD"))

func TestAccPublicCloudPrivateNetwork_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCheckPublicCloudPrivateNetworkPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPublicCloudPrivateNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPublicCloudPrivateNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVRackPublicCloudAttachmentExists("ovh_vrack_publiccloud_attachment.attach", t),
					testAccCheckPublicCloudPrivateNetworkExists("ovh_publiccloud_private_network.network", t),
				),
			},
		},
	})
}

func testAccCheckPublicCloudPrivateNetworkPreCheck(t *testing.T) {
	testAccPreCheck(t)
	testAccCheckPublicCloudExists(t)
}

func testAccCheckPublicCloudPrivateNetworkExists(n string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*Config)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if rs.Primary.Attributes["project_id"] == "" {
			return fmt.Errorf("No Project ID is set")
		}

		return publicCloudPrivateNetworkExists(rs.Primary.Attributes["project_id"], rs.Primary.ID, config.OVHClient)
	}
}

func testAccCheckPublicCloudPrivateNetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ovh_publiccloud_private_network" {
			continue
		}

		err := publicCloudPrivateNetworkExists(rs.Primary.Attributes["project_id"], rs.Primary.ID, config.OVHClient)
		if err == nil {
			return fmt.Errorf("VRack > Public Cloud Private Network still exists")
		}

	}
	return nil
}
