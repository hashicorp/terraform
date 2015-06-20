package softlayer

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

func TestAccSoftLayerVirtualGuest_Basic(t *testing.T) {
	var guest datatypes.SoftLayer_Virtual_Guest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerVirtualGuestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:  testAccCheckSoftLayerVirtualGuestConfig_basic,
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerVirtualGuestExists("softlayer_virtual_guest.terraform-acceptance-test-1", &guest),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "name", "terraform-test"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "domain", "bar.example.com"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "region", "ams01"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "public_network_speed", "10"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "hourly_billing", "true"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "private_network_only", "false"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "cpu", "1"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "ram", "1024"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "disks.0", "25"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "disks.1", "10"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "disks.2", "20"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "user_data", "{\"value\":\"newvalue\"}"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "local_disk", "false"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "dedicated_acct_host_only", "true"),

					// TODO: As agreed, will be enabled when VLAN support is implemented: https://github.com/TheWeatherCompany/softlayer-go/issues/3
					//					resource.TestCheckResourceAttr(
					//						"softlayer_virtual_guest.terraform-acceptance-test-1", "frontend_vlan_id", "1085155"),
					//					resource.TestCheckResourceAttr(
					//						"softlayer_virtual_guest.terraform-acceptance-test-1", "backend_vlan_id", "1085157"),
				),
			},

			resource.TestStep{
				Config:  testAccCheckSoftLayerVirtualGuestConfig_userDataUpdate,
				Destroy: false,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerVirtualGuestExists("softlayer_virtual_guest.terraform-acceptance-test-1", &guest),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "user_data", "updatedData"),
				),
			},

			resource.TestStep{
				Config: testAccCheckSoftLayerVirtualGuestConfig_upgradeMemoryNetworkSpeed,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerVirtualGuestExists("softlayer_virtual_guest.terraform-acceptance-test-1", &guest),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "ram", "2048"),
					resource.TestCheckResourceAttr(
						"softlayer_virtual_guest.terraform-acceptance-test-1", "public_network_speed", "100"),
				),
			},

			// TODO: currently CPU upgrade test is disabled, due to unexpected behavior of field "dedicated_acct_host_only".
			// TODO: For some reason it is reset by SoftLayer to "false". Daniel Bright reported corresponding issue to SoftLayer team.
			//			resource.TestStep{
			//				Config: testAccCheckSoftLayerVirtualGuestConfig_vmUpgradeCPUs,
			//				Check: resource.ComposeTestCheckFunc(
			//					testAccCheckSoftLayerVirtualGuestExists("softlayer_virtual_guest.terraform-acceptance-test-1", &guest),
			//					resource.TestCheckResourceAttr(
			//						"softlayer_virtual_guest.terraform-acceptance-test-1", "cpu", "2"),
			//				),
			//			},

		},
	})
}

func TestAccSoftLayerVirtualGuest_BlockDeviceTemplateGroup(t *testing.T) {
	var guest datatypes.SoftLayer_Virtual_Guest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerVirtualGuestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerVirtualGuestConfig_blockDeviceTemplateGroup,
				Check: resource.ComposeTestCheckFunc(
					// block_device_template_group_gid value is hardcoded. If it's valid then virtual guest will be created well
					testAccCheckSoftLayerVirtualGuestExists("softlayer_virtual_guest.terraform-acceptance-test-BDTGroup", &guest),
				),
			},
		},
	})
}

func TestAccSoftLayerVirtualGuest_postInstallScriptUri(t *testing.T) {
	var guest datatypes.SoftLayer_Virtual_Guest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerVirtualGuestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerVirtualGuestConfig_postInstallScriptUri,
				Check: resource.ComposeTestCheckFunc(
					// block_device_template_group_gid value is hardcoded. If it's valid then virtual guest will be created well
					testAccCheckSoftLayerVirtualGuestExists("softlayer_virtual_guest.terraform-acceptance-test-pISU", &guest),
				),
			},
		},
	})
}

func testAccCheckSoftLayerVirtualGuestDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).virtualGuestService

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "softlayer_virtual_guest" {
			continue
		}

		guestId, _ := strconv.Atoi(rs.Primary.ID)

		// Try to find the guest
		_, err := client.GetObject(guestId)

		// Wait

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for virtual guest (%s) to be destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckSoftLayerVirtualGuestExists(n string, guest *datatypes.SoftLayer_Virtual_Guest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No virtual guest ID is set")
		}

		id, err := strconv.Atoi(rs.Primary.ID)

		if err != nil {
			return err
		}

		client := testAccProvider.Meta().(*Client).virtualGuestService
		retrieveVirtGuest, err := client.GetObject(id)

		if err != nil {
			return err
		}

		fmt.Printf("The ID is %d", id)

		if retrieveVirtGuest.Id != id {
			return fmt.Errorf("Virtual guest not found")
		}

		*guest = retrieveVirtGuest

		return nil
	}
}

const testAccCheckSoftLayerVirtualGuestConfig_basic = `
resource "softlayer_virtual_guest" "terraform-acceptance-test-1" {
    name = "terraform-test"
    domain = "bar.example.com"
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 10
    hourly_billing = true
	private_network_only = false
    cpu = 1
    ram = 1024
    disks = [25, 10, 20]
    user_data = "{\"value\":\"newvalue\"}"
    dedicated_acct_host_only = true
    local_disk = false
}
`

const testAccCheckSoftLayerVirtualGuestConfig_userDataUpdate = `
resource "softlayer_virtual_guest" "terraform-acceptance-test-1" {
    name = "terraform-test"
    domain = "bar.example.com"
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 10
    hourly_billing = true
    cpu = 1
    ram = 1024
    disks = [25, 10, 20]
    user_data = "updatedData"
    dedicated_acct_host_only = true
    local_disk = false
}
`

const testAccCheckSoftLayerVirtualGuestConfig_upgradeMemoryNetworkSpeed = `
resource "softlayer_virtual_guest" "terraform-acceptance-test-1" {
    name = "terraform-test"
    domain = "bar.example.com"
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 100
    hourly_billing = true
    cpu = 1
    ram = 2048
    disks = [25, 10, 20]
    user_data = "updatedData"
    dedicated_acct_host_only = true
    local_disk = false
}
`

const testAccCheckSoftLayerVirtualGuestConfig_vmUpgradeCPUs = `
resource "softlayer_virtual_guest" "terraform-acceptance-test-1" {
    name = "terraform-test"
    domain = "bar.example.com"
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 100
    hourly_billing = true
    cpu = 2
    ram = 2048
    disks = [25, 10, 20]
    user_data = "updatedData"
    dedicated_acct_host_only = true
    local_disk = false
}
`

const testAccCheckSoftLayerVirtualGuestConfig_postInstallScriptUri = `
resource "softlayer_virtual_guest" "terraform-acceptance-test-pISU" {
    name = "terraform-test-pISU"
    domain = "bar.example.com"
    image = "DEBIAN_7_64"
    region = "ams01"
    public_network_speed = 10
    hourly_billing = true
	private_network_only = false
    cpu = 1
    ram = 1024
    disks = [25, 10, 20]
    user_data = "{\"value\":\"newvalue\"}"
    dedicated_acct_host_only = true
    local_disk = false
    post_install_script_uri = "https://www.google.com"
}
`

const testAccCheckSoftLayerVirtualGuestConfig_blockDeviceTemplateGroup = `
resource "softlayer_virtual_guest" "terraform-acceptance-test-BDTGroup" {
    name = "terraform-test-blockDeviceTemplateGroup"
    domain = "bar.example.com"
    region = "ams01"
    public_network_speed = 10
    hourly_billing = false
    cpu = 1
    ram = 1024
    local_disk = false
    block_device_template_group_gid = "ac2b413c-9893-4178-8e62-a24cbe2864db"
}
`
