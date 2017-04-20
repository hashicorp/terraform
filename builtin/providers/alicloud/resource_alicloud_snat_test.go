package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccAlicloudSnat_basic(t *testing.T) {
	var snat ecs.SnatEntrySetType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_snat_entry.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckSnatEntryDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSnatEntryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnatEntryExists(
						"alicloud_snat_entry.foo", &snat),
				),
			},
			resource.TestStep{
				Config: testAccSnatEntryUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnatEntryExists(
						"alicloud_snat_entry.foo", &snat),
				),
			},
		},
	})

}

func testAccCheckSnatEntryDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_snat_entry" {
			continue
		}

		// Try to find the Snat entry
		instance, err := client.DescribeSnatEntry(rs.Primary.Attributes["snat_table_id"], rs.Primary.ID)

		//this special deal cause the DescribeSnatEntry can't find the records would be throw "cant find the snatTable error"
		if instance.SnatEntryId == "" {
			return nil
		}

		if instance.SnatEntryId != "" {
			return fmt.Errorf("Snat entry still exist")
		}

		if err != nil {
			// Verify the error is what we want
			e, _ := err.(*common.Error)

			if !notFoundError(e) {
				return err
			}
		}

	}

	return nil
}

func testAccCheckSnatEntryExists(n string, snat *ecs.SnatEntrySetType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SnatEntry ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		instance, err := client.DescribeSnatEntry(rs.Primary.Attributes["snat_table_id"], rs.Primary.ID)

		if err != nil {
			return err
		}
		if instance.SnatEntryId == "" {
			return fmt.Errorf("SnatEntry not found")
		}

		*snat = instance
		return nil
	}
}

const testAccSnatEntryConfig = `
data "alicloud_zones" "default" {
	"available_resource_creation"= "VSwitch"
}

resource "alicloud_vpc" "foo" {
	name = "tf_test_foo"
	cidr_block = "172.16.0.0/12"
}

resource "alicloud_vswitch" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	cidr_block = "172.16.0.0/21"
	availability_zone = "${data.alicloud_zones.default.zones.2.id}"
}

resource "alicloud_nat_gateway" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	spec = "Small"
	name = "test_foo"
	bandwidth_packages = [{
	  ip_count = 2
	  bandwidth = 5
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	},{
	  ip_count = 1
	  bandwidth = 6
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	}]
	depends_on = [
    	"alicloud_vswitch.foo"]
}
resource "alicloud_snat_entry" "foo"{
	snat_table_id = "${alicloud_nat_gateway.foo.snat_table_ids}"
	source_vswitch_id = "${alicloud_vswitch.foo.id}"
	snat_ip = "${alicloud_nat_gateway.foo.bandwidth_packages.0.public_ip_addresses}"
}
`

const testAccSnatEntryUpdate = `
data "alicloud_zones" "default" {
	"available_resource_creation"= "VSwitch"
}

resource "alicloud_vpc" "foo" {
	name = "tf_test_foo"
	cidr_block = "172.16.0.0/12"
}

resource "alicloud_vswitch" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	cidr_block = "172.16.0.0/21"
	availability_zone = "${data.alicloud_zones.default.zones.2.id}"
}

resource "alicloud_nat_gateway" "foo" {
	vpc_id = "${alicloud_vpc.foo.id}"
	spec = "Small"
	name = "test_foo"
	bandwidth_packages = [{
	  ip_count = 2
	  bandwidth = 5
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	},{
	  ip_count = 1
	  bandwidth = 6
	  zone = "${data.alicloud_zones.default.zones.2.id}"
	}]
	depends_on = [
    	"alicloud_vswitch.foo"]
}
resource "alicloud_snat_entry" "foo"{
	snat_table_id = "${alicloud_nat_gateway.foo.snat_table_ids}"
	source_vswitch_id = "${alicloud_vswitch.foo.id}"
	snat_ip = "${alicloud_nat_gateway.foo.bandwidth_packages.1.public_ip_addresses}"
}
`
