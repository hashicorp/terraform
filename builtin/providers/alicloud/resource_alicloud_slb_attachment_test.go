package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/slb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"testing"
)

func TestAccAlicloudSlbAttachment_basic(t *testing.T) {
	var slb slb.LoadBalancerType

	testCheckAttr := func() resource.TestCheckFunc {
		return func(*terraform.State) error {
			log.Printf("testCheckAttr slb BackendServers is: %#v", slb.BackendServers)
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_slb_attachment.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckSlbDestroy,
		Steps: []resource.TestStep{
			//test internet_charge_type is paybybandwidth
			resource.TestStep{
				Config: testAccSlbAttachment,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSlbExists("alicloud_slb_attachment.foo", &slb),
					testCheckAttr(),
					testAccCheckAttachment("alicloud_instance.foo", &slb),
				),
			},
		},
	})
}

func testAccCheckAttachment(n string, slb *slb.LoadBalancerType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ECS ID is set")
		}

		ecsInstanceId := rs.Primary.ID

		backendServers := slb.BackendServers.BackendServer

		if len(backendServers) == 0 {
			return fmt.Errorf("no SLB backendServer: %#v", backendServers)
		}

		log.Printf("slb bacnendservers: %#v", backendServers)

		backendServersInstanceId := backendServers[0].ServerId

		if ecsInstanceId != backendServersInstanceId {
			return fmt.Errorf("SLB attachment check invalid: ECS instance %s is not equal SLB backendServer %s",
				ecsInstanceId, backendServersInstanceId)
		}
		return nil
	}
}

const testAccSlbAttachment = `
resource "alicloud_security_group" "foo" {
	name = "tf_test_foo"
	description = "foo"
}

resource "alicloud_security_group_rule" "http-in" {
  	type = "ingress"
  	ip_protocol = "tcp"
  	nic_type = "internet"
  	policy = "accept"
  	port_range = "80/80"
  	priority = 1
  	security_group_id = "${alicloud_security_group.foo.id}"
  	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_security_group_rule" "ssh-in" {
  	type = "ingress"
  	ip_protocol = "tcp"
  	nic_type = "internet"
  	policy = "accept"
  	port_range = "22/22"
  	priority = 1
  	security_group_id = "${alicloud_security_group.foo.id}"
  	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_instance" "foo" {
	# cn-beijing
	image_id = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"

	# series II
	instance_type = "ecs.n1.medium"
	internet_charge_type = "PayByBandwidth"
	internet_max_bandwidth_out = "5"
	system_disk_category = "cloud_efficiency"
	io_optimized = "optimized"

	security_groups = ["${alicloud_security_group.foo.id}"]
	instance_name = "test_foo"
}

resource "alicloud_slb" "foo" {
	name = "tf_test_slb_bind"
	internet_charge_type = "paybybandwidth"
	bandwidth = "5"
	internet = "true"
}

resource "alicloud_slb_attachment" "foo" {
	slb_id = "${alicloud_slb.foo.id}"
	instances = ["${alicloud_instance.foo.id}"]
}

`
