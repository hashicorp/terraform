package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ess"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"testing"
)

func TestAccAlicloudEssScalingGroup_basic(t *testing.T) {
	var sg ess.ScalingGroupItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_group.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingGroupExists(
						"alicloud_ess_scaling_group.foo", &sg),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"min_size",
						"1"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"max_size",
						"1"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"scaling_group_name",
						"foo"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"removal_policies.#",
						"2",
					),
				),
			},
		},
	})

}

func TestAccAlicloudEssScalingGroup_update(t *testing.T) {
	var sg ess.ScalingGroupItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_group.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingGroupExists(
						"alicloud_ess_scaling_group.foo", &sg),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"min_size",
						"1"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"max_size",
						"1"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"scaling_group_name",
						"foo"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"removal_policies.#",
						"2",
					),
				),
			},

			resource.TestStep{
				Config: testAccEssScalingGroup_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingGroupExists(
						"alicloud_ess_scaling_group.foo", &sg),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"min_size",
						"2"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"max_size",
						"2"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"scaling_group_name",
						"update"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"removal_policies.#",
						"1",
					),
				),
			},
		},
	})

}

func SkipTestAccAlicloudEssScalingGroup_vpc(t *testing.T) {
	var sg ess.ScalingGroupItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_group.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingGroup_vpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingGroupExists(
						"alicloud_ess_scaling_group.foo", &sg),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"min_size",
						"1"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"max_size",
						"1"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"scaling_group_name",
						"foo"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_group.foo",
						"removal_policies.#",
						"2",
					),
				),
			},
		},
	})

}

func testAccCheckEssScalingGroupExists(n string, d *ess.ScalingGroupItemType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ESS Scaling Group ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		attr, err := client.DescribeScalingGroupById(rs.Primary.ID)
		log.Printf("[DEBUG] check scaling group %s attribute %#v", rs.Primary.ID, attr)

		if err != nil {
			return err
		}

		if attr == nil {
			return fmt.Errorf("Scaling Group not found")
		}

		*d = *attr
		return nil
	}
}

func testAccCheckEssScalingGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_ess_scaling_group" {
			continue
		}

		ins, err := client.DescribeScalingGroupById(rs.Primary.ID)

		if ins != nil {
			return fmt.Errorf("Error ESS scaling group still exist")
		}

		// Verify the error is what we want
		if err != nil {
			// Verify the error is what we want
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == InstanceNotfound {
				continue
			}
			return err
		}
	}

	return nil
}

const testAccEssScalingGroupConfig = `
resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}
`

const testAccEssScalingGroup = `
resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}
`

const testAccEssScalingGroup_update = `
resource "alicloud_ess_scaling_group" "foo" {
	min_size = 2
	max_size = 2
	scaling_group_name = "update"
	removal_policies = ["OldestInstance"]
}
`
const testAccEssScalingGroup_vpc = `
data "alicloud_images" "ecs_image" {
  most_recent = true
  name_regex =  "^centos_6\\w{1,5}[64].*"
}

data "alicloud_zones" "default" {
	"available_disk_category"= "cloud_efficiency"
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

resource "alicloud_security_group" "tf_test_foo" {
	description = "foo"
	vpc_id = "${alicloud_vpc.foo.id}"
}

resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	default_cooldown = 20
	vswitch_id = "${alicloud_vswitch.foo.id}"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"
	enable = true

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.n1.medium"
	io_optimized = "optimized"
	system_disk_category = "cloud_efficiency"
	internet_charge_type = "PayByTraffic"
	internet_max_bandwidth_out = 10
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}
`
