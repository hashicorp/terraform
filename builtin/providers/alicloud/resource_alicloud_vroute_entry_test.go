package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"strings"
	"testing"
)

func TestAccAlicloudRouteEntry_Basic(t *testing.T) {
	var rt ecs.RouteTableSetType
	var rn ecs.RouteEntrySetType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_route_entry.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckRouteEntryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRouteEntryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRouteTableEntryExists(
						"alicloud_route_entry.foo", &rt, &rn),
					resource.TestCheckResourceAttrSet(
						"alicloud_route_entry.foo", "nexthop_id"),
				),
			},
		},
	})

}

func testAccCheckRouteTableExists(rtId string, t *ecs.RouteTableSetType) error {
	client := testAccProvider.Meta().(*AliyunClient)
	//query route table
	rt, terr := client.QueryRouteTableById(rtId)

	if terr != nil {
		return terr
	}

	if rt == nil {
		return fmt.Errorf("Route Table not found")
	}

	*t = *rt
	return nil
}

func testAccCheckRouteEntryExists(routeTableId, cidrBlock, nextHopType, nextHopId string, e *ecs.RouteEntrySetType) error {
	client := testAccProvider.Meta().(*AliyunClient)
	//query route table entry
	re, rerr := client.QueryRouteEntry(routeTableId, cidrBlock, nextHopType, nextHopId)

	if rerr != nil {
		return rerr
	}

	if re == nil {
		return fmt.Errorf("Route Table Entry not found")
	}

	*e = *re
	return nil
}

func testAccCheckRouteTableEntryExists(n string, t *ecs.RouteTableSetType, e *ecs.RouteEntrySetType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Route Entry ID is set")
		}

		parts := strings.Split(rs.Primary.ID, ":")

		//query route table
		err := testAccCheckRouteTableExists(parts[0], t)

		if err != nil {
			return err
		}
		//query route table entry
		err = testAccCheckRouteEntryExists(parts[0], parts[2], parts[3], parts[4], e)
		return err
	}
}

func testAccCheckRouteEntryDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_route_entry" {
			continue
		}

		parts := strings.Split(rs.Primary.ID, ":")
		re, err := client.QueryRouteEntry(parts[0], parts[2], parts[3], parts[4])

		if re != nil {
			return fmt.Errorf("Error Route Entry still exist")
		}

		// Verify the error is what we want
		if err != nil {
			if notFoundError(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

const testAccRouteEntryConfig = `
data "alicloud_zones" "default" {
	"available_resource_creation"= "VSwitch"
}

resource "alicloud_vpc" "foo" {
	name = "tf_test_foo"
	cidr_block = "10.1.0.0/21"
}

resource "alicloud_vswitch" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	cidr_block = "10.1.1.0/24"
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
}

resource "alicloud_route_entry" "foo" {
	router_id = "${alicloud_vpc.foo.router_id}"
	route_table_id = "${alicloud_vpc.foo.router_table_id}"
	destination_cidrblock = "172.11.1.1/32"
	nexthop_type = "Instance"
	nexthop_id = "${alicloud_instance.foo.id}"
}

resource "alicloud_security_group" "tf_test_foo" {
	name = "tf_test_foo"
	description = "foo"
	vpc_id = "${alicloud_vpc.foo.id}"
}

resource "alicloud_security_group_rule" "ingress" {
	type = "ingress"
	ip_protocol = "tcp"
	nic_type = "intranet"
	policy = "accept"
	port_range = "22/22"
	priority = 1
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_instance" "foo" {
	# cn-beijing
	security_groups = ["${alicloud_security_group.tf_test_foo.id}"]

	vswitch_id = "${alicloud_vswitch.foo.id}"
	allocate_public_ip = true

	# series II
	instance_charge_type = "PostPaid"
	instance_type = "ecs.n1.small"
	internet_charge_type = "PayByTraffic"
	internet_max_bandwidth_out = 5
	io_optimized = "optimized"

	system_disk_category = "cloud_efficiency"
	image_id = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
	instance_name = "test_foo"
}

`
