package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ess"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"regexp"
	"strings"
	"testing"
)

func TestAccAlicloudEssScalingConfiguration_basic(t *testing.T) {
	var sc ess.ScalingConfigurationItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_configuration.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingConfigurationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingConfigurationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingConfigurationExists(
						"alicloud_ess_scaling_configuration.foo", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"instance_type",
						"ecs.s2.large"),
					resource.TestMatchResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"image_id",
						regexp.MustCompile("^centos_6")),
				),
			},
		},
	})
}

func TestAccAlicloudEssScalingConfiguration_multiConfig(t *testing.T) {
	var sc ess.ScalingConfigurationItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_configuration.bar",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingConfigurationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingConfiguration_multiConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingConfigurationExists(
						"alicloud_ess_scaling_configuration.bar", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"active",
						"false"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"instance_type",
						"ecs.s2.large"),
					resource.TestMatchResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"image_id",
						regexp.MustCompile("^centos_6")),
				),
			},
		},
	})
}

func SkipTestAccAlicloudEssScalingConfiguration_active(t *testing.T) {
	var sc ess.ScalingConfigurationItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_configuration.bar",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingConfigurationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingConfiguration_active,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingConfigurationExists(
						"alicloud_ess_scaling_configuration.bar", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"active",
						"true"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"instance_type",
						"ecs.s2.large"),
					resource.TestMatchResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"image_id",
						regexp.MustCompile("^centos_6")),
				),
			},

			resource.TestStep{
				Config: testAccEssScalingConfiguration_inActive,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingConfigurationExists(
						"alicloud_ess_scaling_configuration.bar", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"active",
						"false"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"instance_type",
						"ecs.s2.large"),
					resource.TestMatchResourceAttr(
						"alicloud_ess_scaling_configuration.bar",
						"image_id",
						regexp.MustCompile("^centos_6")),
				),
			},
		},
	})
}

func SkipTestAccAlicloudEssScalingConfiguration_enable(t *testing.T) {
	var sc ess.ScalingConfigurationItemType

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: "alicloud_ess_scaling_configuration.foo",

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingConfigurationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEssScalingConfiguration_enable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingConfigurationExists(
						"alicloud_ess_scaling_configuration.foo", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"enable",
						"true"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"instance_type",
						"ecs.s2.large"),
					resource.TestMatchResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"image_id",
						regexp.MustCompile("^centos_6")),
				),
			},

			resource.TestStep{
				Config: testAccEssScalingConfiguration_disable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEssScalingConfigurationExists(
						"alicloud_ess_scaling_configuration.foo", &sc),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"enable",
						"false"),
					resource.TestCheckResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"instance_type",
						"ecs.s2.large"),
					resource.TestMatchResourceAttr(
						"alicloud_ess_scaling_configuration.foo",
						"image_id",
						regexp.MustCompile("^centos_6")),
				),
			},
		},
	})
}

func testAccCheckEssScalingConfigurationExists(n string, d *ess.ScalingConfigurationItemType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ESS Scaling Configuration ID is set")
		}

		client := testAccProvider.Meta().(*AliyunClient)
		ids := strings.Split(rs.Primary.ID, COLON_SEPARATED)
		attr, err := client.DescribeScalingConfigurationById(ids[0], ids[1])
		log.Printf("[DEBUG] check scaling configuration %s attribute %#v", rs.Primary.ID, attr)

		if err != nil {
			return err
		}

		if attr == nil {
			return fmt.Errorf("Scaling Configuration not found")
		}

		*d = *attr
		return nil
	}
}

func testAccCheckEssScalingConfigurationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_ess_scaling_configuration" {
			continue
		}
		ids := strings.Split(rs.Primary.ID, COLON_SEPARATED)
		ins, err := client.DescribeScalingConfigurationById(ids[0], ids[1])

		if ins != nil {
			return fmt.Errorf("Error ESS scaling configuration still exist")
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

const testAccEssScalingConfigurationConfig = `
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

resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}
`

const testAccEssScalingConfiguration_multiConfig = `
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

resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}

resource "alicloud_ess_scaling_configuration" "bar" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}
`

const testAccEssScalingConfiguration_active = `
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

resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"
	active = true

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}
`

const testAccEssScalingConfiguration_inActive = `
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

resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"
	active = false

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}
`

const testAccEssScalingConfiguration_enable = `
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

resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"
	enable = true

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}
`

const testAccEssScalingConfiguration_disable = `
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

resource "alicloud_ess_scaling_group" "foo" {
	min_size = 1
	max_size = 1
	scaling_group_name = "foo"
	removal_policies = ["OldestInstance", "NewestInstance"]
}

resource "alicloud_ess_scaling_configuration" "foo" {
	scaling_group_id = "${alicloud_ess_scaling_group.foo.id}"
	enable = false

	image_id = "${data.alicloud_images.ecs_image.images.0.id}"
	instance_type = "ecs.s2.large"
	io_optimized = "optimized"
	security_group_id = "${alicloud_security_group.tf_test_foo.id}"
}
`
