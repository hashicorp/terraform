package softlayer

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSoftLayerVirtualIpAddress_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSoftLayerVirtualIpAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerVirtualIpAddressConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_virtualipaddress.testacc_vip", "load_balancing_method", "lc"),
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_virtualipaddress.testacc_vip", "name", "test_load_balancer_vip"),
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_virtualipaddress.testacc_vip", "source_port", "80"),
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_virtualipaddress.testacc_vip", "type", "HTTP"),
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_virtualipaddress.testacc_vip", "virtual_ip_address", "23.246.204.65"),
				),
			},
		},
	})
}

func testAccCheckSoftLayerVirtualIpAddressDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).networkApplicationDeliveryControllerService

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "softlayer_network_loadbalancer_virtualipaddress" {
			continue
		}

		nadcId, _ := strconv.Atoi(rs.Primary.Attributes["nad_controller_id"])
		vipName, _ := rs.Primary.Attributes["name"]

		// Try to find the vip
		result, err := client.GetVirtualIpAddress(nadcId, vipName)

		if err != nil {
			return fmt.Errorf("Error fetching virtual ip")
		}

		if len(result.VirtualIpAddress) != 0 {
			return fmt.Errorf("Virtual ip address still exists")
		}
	}

	return nil
}

var testAccCheckSoftLayerVirtualIpAddressConfig_basic = `
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

resource "softlayer_network_application_delivery_controller" "testacc_foobar_nadc" {
    datacenter = "DALLAS05"
    speed = 10
    version = "10.1"
    plan = "Standard"
    ip_count = 2
}

resource "softlayer_network_loadbalancer_virtualipaddress" "testacc_vip" {
    name = "test_load_balancer_vip"
    nad_controller_id = "${softlayer_network_application_delivery_controller.testacc_foobar_nadc.id}"
    load_balancing_method = "lc"
    source_port = 80
    type = "HTTP"
    virtual_ip_address = "${softlayer_virtual_guest.terraform-acceptance-test-1.ipv4_address}"
}
`
