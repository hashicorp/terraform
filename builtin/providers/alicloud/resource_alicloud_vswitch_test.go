package alicloud

import (
	"testing"

	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAlicloudVswitch_basic(t *testing.T) {
	var vsw ecs.VSwitchSetType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_vswitch.foo",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckVswitchDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVswitchConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVswitchExists("alicloud_vswitch.foo", &vsw),
					resource.TestCheckResourceAttr(
						"alicloud_vswitch.foo", "cidr_block", "172.16.0.0/21"),
				),
			},
		},
	})

}

func testAccCheckVswitchExists(n string, vpc *ecs.VSwitchSetType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Vswitch ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		instance, err := client.QueryVswitchById(rs.Primary.Attributes["vpc_id"], rs.Primary.ID)

		if err != nil {
			return err
		}
		if instance == nil {
			return fmt.Errorf("Vswitch not found")
		}

		*vpc = *instance
		return nil
	}
}

func testAccCheckVswitchDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_vswitch" {
			continue
		}

		// Try to find the Vswitch
		instance, err := client.QueryVswitchById(rs.Primary.Attributes["vpc_id"], rs.Primary.ID)

		if instance != nil {
			return fmt.Errorf("Vswitch still exist")
		}

		if err != nil {
			// Verify the error is what we want
			e, _ := err.(*common.Error)

			if e.ErrorResponse.Code != "InvalidVswitchID.NotFound" {
				return err
			}
		}

	}

	return nil
}

const testAccVswitchConfig = `
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
  availability_zone = "${data.alicloud_zones.default.zones.0.id}"
}
`
