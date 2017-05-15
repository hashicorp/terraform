package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var testAccPublicCloudPrivateNetworkSubnetConfig = fmt.Sprintf(`
resource "ovh_vrack_publiccloud_attachment" "attach" {
  vrack_id   = "%s"
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
  vlan_id     = 0
  name        = "terraform_testacc_private_net"
  regions     = ["${data.ovh_publiccloud_regions.regions.names}"]
}

resource "ovh_publiccloud_private_network_subnet" "subnet" {
  project_id = "${ovh_publiccloud_private_network.network.project_id}"
  network_id = "${ovh_publiccloud_private_network.network.id}"
  region     = "${element(data.ovh_publiccloud_regions.regions.names, 0)}"
  start      = "192.168.168.100"
  end        = "192.168.168.200"
  network    = "192.168.168.0/24"
  dhcp       = true
  no_gateway = false
}
`, os.Getenv("OVH_VRACK"), os.Getenv("OVH_PUBLIC_CLOUD"))

func TestAccPublicCloudPrivateNetworkSubnet_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccCheckPublicCloudPrivateNetworkSubnetPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPublicCloudPrivateNetworkSubnetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPublicCloudPrivateNetworkSubnetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVRackPublicCloudAttachmentExists("ovh_vrack_publiccloud_attachment.attach", t),
					testAccCheckPublicCloudPrivateNetworkExists("ovh_publiccloud_private_network.network", t),
					testAccCheckPublicCloudPrivateNetworkSubnetExists("ovh_publiccloud_private_network_subnet.subnet", t),
				),
			},
		},
	})
}

func testAccCheckPublicCloudPrivateNetworkSubnetPreCheck(t *testing.T) {
	testAccPreCheck(t)
	testAccCheckPublicCloudExists(t)
}

func testAccCheckPublicCloudPrivateNetworkSubnetExists(n string, t *testing.T) resource.TestCheckFunc {
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

		if rs.Primary.Attributes["network_id"] == "" {
			return fmt.Errorf("No Network ID is set")
		}

		return publicCloudPrivateNetworkSubnetExists(
			rs.Primary.Attributes["project_id"],
			rs.Primary.Attributes["network_id"],
			rs.Primary.ID,
			config.OVHClient,
		)
	}
}

func testAccCheckPublicCloudPrivateNetworkSubnetDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ovh_publiccloud_private_network_subnet" {
			continue
		}

		err := publicCloudPrivateNetworkSubnetExists(
			rs.Primary.Attributes["project_id"],
			rs.Primary.Attributes["network_id"],
			rs.Primary.ID,
			config.OVHClient,
		)

		if err == nil {
			return fmt.Errorf("VRack > Public Cloud Private Network Subnet still exists")
		}

	}
	return nil
}
