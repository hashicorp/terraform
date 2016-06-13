package spotinst

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

func TestAccSpotinstGroup_Basic(t *testing.T) {
	var group spotinst.AwsGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSpotinstGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSpotinstGroupConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstGroupExists("spotinst_aws_group.foo", &group), testAccCheckSpotinstGroupAttributes(&group),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "name", "terraform"),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "description", "terraform"),
				),
			},
		},
	})
}

func TestAccSpotinstGroup_Updated(t *testing.T) {
	var group spotinst.AwsGroup
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSpotinstGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSpotinstGroupConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstGroupExists("spotinst_aws_group.foo", &group), testAccCheckSpotinstGroupAttributes(&group),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "name", "terraform"),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "description", "terraform"),
				),
			},
			resource.TestStep{
				Config: testAccCheckSpotinstGroupConfigNewValue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstGroupExists("spotinst_aws_group.foo", &group), testAccCheckSpotinstGroupAttributesUpdated(&group),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "name", "terraform_updated"),
					resource.TestCheckResourceAttr("spotinst_aws_group.foo", "description", "terraform_updated"),
				),
			},
		},
	})
}

func testAccCheckSpotinstGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*spotinst.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "spotinst_aws_group" {
			continue
		}

		_, _, err := client.AwsGroup.Get(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Group still exists")
		}
	}

	return nil
}

func testAccCheckSpotinstGroupAttributes(group *spotinst.AwsGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.Name != "terraform" {
			return fmt.Errorf("Bad content: %v", group.Name)
		}

		return nil
	}
}

func testAccCheckSpotinstGroupAttributesUpdated(group *spotinst.AwsGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *group.Name != "terraform_updated" {
			return fmt.Errorf("Bad content: %v", group.Name)
		}

		return nil
	}
}

func testAccCheckSpotinstGroupExists(n string, group *spotinst.AwsGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}

		client := testAccProvider.Meta().(*spotinst.Client)
		foundGroups, _, err := client.AwsGroup.Get(rs.Primary.ID)

		if err != nil {
			return err
		}

		if *foundGroups[0].Name != rs.Primary.Attributes["name"] {
			return fmt.Errorf("Group not found: %+v,\n %+v\n", foundGroups[0], rs.Primary.Attributes)
		}

		*group = *foundGroups[0]

		return nil
	}
}

const testAccCheckSpotinstGroupConfigBasic = `
resource "spotinst_aws_group" "foo" {
	name = "terraform"
	description = "terraform"
	product = "Linux/UNIX"

	capacity {
		target = 0
		minimum = 0
		maximum = 5
	}

	strategy {
		risk = 100
	}

	instance_types {
		ondemand = "c3.large"
		spot = ["c3.large"]
	}

	availability_zone {
		name = "us-west-2b"
	}

	signal {
		name = "instance_ready"
	}

	launch_specification {
		monitoring = false
		image_id = "ami-f0091d91"
		key_pair = "east"
		security_group_ids = ["default"]
		user_data = "#!/bin/sh echo hello"
	}

	ebs_block_device {
		device_name = "foo"
		delete_on_termination = false
	}

	ebs_block_device {
		device_name = "bar"
		delete_on_termination = true
	}

	ephemeral_block_device {
		device_name = "baz"
		virtual_name = "xvda"
	}

	network_interface {
		description = "foo"
		device_index = 1
		secondary_private_ip_address_count = 1
		associate_public_ip_address = false
		delete_on_termination = false
		security_group_ids = ["foo"]
		network_interface_id = "bar"
		private_ip_address = "172.0.0.1"
		subnet_id = "foo"
	}

	elastic_ips = [
		"eipalloc-01",
		"eipalloc-02"
	]

	tags {
		foo = "bar"
		bar = "baz"
	}

	scaling_up_policy {
		policy_name = "Scaling Policy 1"
		metric_name = "CPUUtilization"
		statistic = "average"
		unit = "percent"
		threshold = 80
		adjustment = 1
		namespace = "AWS/EC2"
		period = 300
		evaluation_periods = 2
		cooldown = 300
	}

	scaling_down_policy {
		policy_name = "Scaling Policy 2"
		metric_name = "CPUUtilization"
		statistic = "average"
		unit = "percent"
		threshold = 40
		adjustment = 1
		namespace = "AWS/EC2"
		period = 300
		evaluation_periods = 2
		cooldown = 300
	}

	scheduled_task {
		task_type = "scale"
		cron_expression = "0 5 * * 0-4"
		scale_target_capacity = 2
	}

	scheduled_task {
		task_type = "scale"
		cron_expression = "0 20 * * 0-4"
		scale_target_capacity = 0
	}

	scheduled_task {
		task_type = "backup_ami"
		frequency = "hourly"
	}

	rancher_integration {
		master_host = "localhost"
		access_key = "foo"
		secret_key = "bar"
	}
}`

const testAccCheckSpotinstGroupConfigNewValue = `
resource "spotinst_aws_group" "foo" {
	name = "terraform_updated"
	description = "terraform_updated"
	product = "Linux/UNIX"

	capacity {
		target = 0
		minimum = 0
		maximum = 5
	}

	strategy {
		risk = 100
	}

	instance_types {
		ondemand = "c3.large"
		spot = ["c3.large"]
	}

	availability_zone {
		name = "us-west-2b"
	}

	signal {
		name = "instance_ready"
	}

	launch_specification {
		monitoring = false
		image_id = "ami-f0091d91"
		key_pair = "east"
		security_group_ids = ["default"]
		user_data = "#!/bin/sh echo hello"
	}

	ebs_block_device {
		device_name = "foo"
		delete_on_termination = false
	}

	ebs_block_device {
		device_name = "bar"
		delete_on_termination = true
	}

	ephemeral_block_device {
		device_name = "baz"
		virtual_name = "xvda"
	}

	network_interface {
		description = "foo"
		device_index = 1
		secondary_private_ip_address_count = 1
		associate_public_ip_address = false
		delete_on_termination = false
		security_group_ids = ["foo"]
		network_interface_id = "bar"
		private_ip_address = "172.0.0.1"
		subnet_id = "foo"
	}

	elastic_ips = [
		"eipalloc-01",
		"eipalloc-02"
	]

	tags {
		foo = "bar"
		bar = "baz"
	}

	scaling_up_policy {
		policy_name = "Scaling Policy 1"
		metric_name = "CPUUtilization"
		statistic = "average"
		unit = "percent"
		threshold = 80
		adjustment = 1
		namespace = "AWS/EC2"
		period = 300
		evaluation_periods = 2
		cooldown = 300
	}

	scaling_down_policy {
		policy_name = "Scaling Policy 2"
		metric_name = "CPUUtilization"
		statistic = "average"
		unit = "percent"
		threshold = 40
		adjustment = 1
		namespace = "AWS/EC2"
		period = 300
		evaluation_periods = 2
		cooldown = 300
	}

	scheduled_task {
		task_type = "scale"
		cron_expression = "0 5 * * 0-4"
		scale_target_capacity = 2
	}

	scheduled_task {
		task_type = "scale"
		cron_expression = "0 20 * * 0-4"
		scale_target_capacity = 0
	}

	scheduled_task {
		task_type = "backup_ami"
		frequency = "hourly"
	}

	rancher_integration {
		master_host = "localhost"
		access_key = "foo"
		secret_key = "bar"
	}
}`
