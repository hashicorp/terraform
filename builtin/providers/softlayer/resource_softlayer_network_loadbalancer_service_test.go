package softlayer

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccSoftLayerLoadBalancerService_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSoftLayerLoadBalancerServiceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_service.testacc_service", "name", "test_load_balancer_service"),
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_service.testacc_service", "destination_port", "89"),
					resource.TestCheckResourceAttr(
						"softlayer_network_loadbalancer_service.testacc_service", "weight", "55"),
				),
			},
		},
	})
}

var testAccCheckSoftLayerLoadBalancerServiceConfig_basic = `
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

resource "softlayer_virtual_guest" "terraform-acceptance-test-2" {
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
    datacenter = "DALLAS06"
    speed = 10
    version = "10.1"
    plan = "Standard"
    ip_count = 2
}

resource "softlayer_network_loadbalancer_virtualipaddress" "testacc_vip" {
    name = "test_load_balancer_vip"
    nad_controller_id = "${softlayer_network_application_delivery_controller.testacc_foobar_nadc.id}"
    load_balancing_method = "lc"
    source_port = 81
    type = "HTTP"
    virtual_ip_address = "${softlayer_virtual_guest.terraform-acceptance-test-1.ipv4_address}"
}

resource "softlayer_network_loadbalancer_service" "testacc_service" {
  name = "test_load_balancer_service"
  vip_id = "${softlayer_network_loadbalancer_virtualipaddress.testacc_vip.id}"
  destination_ip_address = "${softlayer_virtual_guest.terraform-acceptance-test-2.ipv4_address}"
  destination_port = 89
  weight = 55
  connection_limit = 5000
  health_check = "HTTP"
}
`
