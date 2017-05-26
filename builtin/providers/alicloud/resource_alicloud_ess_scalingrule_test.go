package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ess"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"strings"
	"testing"
)

func TestAccAlicloudEssScalingRule_basic(t *testing.T) {
	var sc ess.ScalingRuleItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_rule.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingRuleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingRuleExists(
						"alicloud_ess_scaling_rule.foo", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_rule.foo",
						"adjustment_type",
						"TotalCapacity"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_rule.foo",
						"adjustment_value",
						"1"),
				),
			},
		},
	})
}

func TestAccAlicloudEssScalingRule_update(t *testing.T) {
	var sc ess.ScalingRuleItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_rule.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingRule,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingRuleExists(
						"alicloud_ess_scaling_rule.foo", &sc),
					testAccCheckEssScalingRuleExists(
						"alicloud_ess_scaling_rule.foo", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_rule.foo",
						"adjustment_type",
						"TotalCapacity"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_rule.foo",
						"adjustment_value",
						"1"),
				),
			},

			resource.TestStep{
				Config: testAccEssScalingRule_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingRuleExists(
						"alicloud_ess_scaling_rule.foo", &sc),
					testAccCheckEssScalingRuleExists(
						"alicloud_ess_scaling_rule.foo", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_rule.foo",
						"adjustment_type",
						"TotalCapacity"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_rule.foo",
						"adjustment_value",
						"2"),
				),
			},
		},
	})
}

func testAccCheckEssScalingRuleExists(n string, d *ess.ScalingRuleItemType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ESS Scaling Rule ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		ids := strings.Split(rs.Primary.ID, COLON_SEPARATED)
		attr, err := client.DescribeScalingRuleById(ids[0], ids[1])
		log.Printf("[DEBUG] check scaling rule %s attribute %#v", rs.Primary.ID, attr)

		if err != nil {
			return err
		}

		if attr == nil {
			return fmt.Errorf("Scaling rule not found")
		}

		*d = *attr
		return nil
	}
}

func testAccCheckEssScalingRuleDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_ess_scaling_rule" {
			continue
		}
		ids := strings.Split(rs.Primary.ID, COLON_SEPARATED)
		ins, err := client.DescribeScalingRuleById(ids[0], ids[1])

		if ins != nil {
			return fmt.Errorf("Error ESS scaling rule still exist")
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

const testAccEssScalingRuleConfig = `
data "alicloud_images" "ecs_image" {
  most_recent = true
  name_regex =  "^centos_6\\w{1,5}[64].*"
}

resource "alicloud_security_group" "tf_test_foo" {
	description = "foo"
}

resource "alicloud_security_group_rule" "ssh-in" {
  	type = "ingress"
  	ip_protocol = "tcp"
  	nic_type = "internet"
  	policy = "accept"
  	port_range = "22/22"
  	priority = 1
  	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
  	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_ess_scaling_group" "bar" {
	min_size = 1
	max_size = 1
	scaling_group_name = "bar"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.bar.id}"

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}

resource "alicloud_ess_scaling_rule" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.bar.id}"
	adjustment_type = "TotalCapacity"
	adjustment_value = 1
	cooldown = 120
}
`

const testAccEssScalingRule = `
data "alicloud_images" "ecs_image" {
  most_recent = true
  name_regex =  "^centos_6\\w{1,5}[64].*"
}

resource "alicloud_security_group" "tf_test_foo" {
	description = "foo"
}

resource "alicloud_security_group_rule" "ssh-in" {
  	type = "ingress"
  	ip_protocol = "tcp"
  	nic_type = "internet"
  	policy = "accept"
  	port_range = "22/22"
  	priority = 1
  	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
  	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_ess_scaling_group" "bar" {
	min_size = 1
	max_size = 1
	scaling_group_name = "bar"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.bar.id}"

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}

resource "alicloud_ess_scaling_rule" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.bar.id}"
	adjustment_type = "TotalCapacity"
	adjustment_value = 1
	cooldown = 120
}
`

const testAccEssScalingRule_update = `
data "alicloud_images" "ecs_image" {
  most_recent = true
  name_regex =  "^centos_6\\w{1,5}[64].*"
}

resource "alicloud_security_group" "tf_test_foo" {
	description = "foo"
}

resource "alicloud_security_group_rule" "ssh-in" {
  	type = "ingress"
  	ip_protocol = "tcp"
  	nic_type = "internet"
  	policy = "accept"
  	port_range = "22/22"
  	priority = 1
  	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
  	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_ess_scaling_group" "bar" {
	min_size = 1
	max_size = 1
	scaling_group_name = "bar"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.bar.id}"

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}

resource "alicloud_ess_scaling_rule" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.bar.id}"
	adjustment_type = "TotalCapacity"
	adjustment_value = 2
	cooldown = 60
}
`
