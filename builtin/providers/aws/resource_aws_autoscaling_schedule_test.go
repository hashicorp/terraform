package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAutoscalingSchedule_basic(t *testing.T) {
	var schedule autoscaling.ScheduledUpdateGroupAction

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingScheduleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalingScheduleExists("aws_autoscaling_schedule.foobar", &schedule),
				),
			},
		},
	})
}

func TestAccAWSAutoscalingSchedule_recurrence(t *testing.T) {
	var schedule autoscaling.ScheduledUpdateGroupAction

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingScheduleConfig_recurrence,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalingScheduleExists("aws_autoscaling_schedule.foobar", &schedule),
					resource.TestCheckResourceAttr("aws_autoscaling_schedule.foobar", "recurrence", "0 8 * * *"),
				),
			},
		},
	})
}

func TestAccAWSAutoscalingSchedule_zeroValues(t *testing.T) {
	var schedule autoscaling.ScheduledUpdateGroupAction

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAutoscalingScheduleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingScheduleConfig_zeroValues,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalingScheduleExists("aws_autoscaling_schedule.foobar", &schedule),
				),
			},
		},
	})
}

func testAccCheckScalingScheduleExists(n string, policy *autoscaling.ScheduledUpdateGroupAction) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		autoScalingGroup, _ := rs.Primary.Attributes["autoscaling_group_name"]
		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		params := &autoscaling.DescribeScheduledActionsInput{
			AutoScalingGroupName: aws.String(autoScalingGroup),
			ScheduledActionNames: []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeScheduledActions(params)
		if err != nil {
			return err
		}
		if len(resp.ScheduledUpdateGroupActions) == 0 {
			return fmt.Errorf("Scaling Schedule not found")
		}

		return nil
	}
}

func testAccCheckAWSAutoscalingScheduleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscaling_schedule" {
			continue
		}

		autoScalingGroup, _ := rs.Primary.Attributes["autoscaling_group_name"]
		params := &autoscaling.DescribeScheduledActionsInput{
			AutoScalingGroupName: aws.String(autoScalingGroup),
			ScheduledActionNames: []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeScheduledActions(params)

		if err == nil {
			if len(resp.ScheduledUpdateGroupActions) != 0 &&
				*resp.ScheduledUpdateGroupActions[0].ScheduledActionName == rs.Primary.ID {
				return fmt.Errorf("Scaling Schedule Still Exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccAWSAutoscalingScheduleConfig = fmt.Sprintf(`
resource "aws_launch_configuration" "foobar" {
    name = "terraform-test-foobar5"
    image_id = "ami-21f78e11"
    instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "foobar" {
    availability_zones = ["us-west-2a"]
    name = "terraform-test-foobar5"
    max_size = 1
    min_size = 1
    health_check_grace_period = 300
    health_check_type = "ELB"
    force_delete = true
    termination_policies = ["OldestInstance"]
    launch_configuration = "${aws_launch_configuration.foobar.name}"
    tag {
        key = "Foo"
        value = "foo-bar"
        propagate_at_launch = true
    }
}

resource "aws_autoscaling_schedule" "foobar" {
    scheduled_action_name = "foobar"
    min_size = 0
    max_size = 1
    desired_capacity = 0
    start_time = "2016-12-11T18:00:00Z"
    end_time = "2016-12-12T06:00:00Z"
    autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
`)

var testAccAWSAutoscalingScheduleConfig_recurrence = fmt.Sprintf(`
resource "aws_launch_configuration" "foobar" {
    name = "terraform-test-foobar5"
    image_id = "ami-21f78e11"
    instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "foobar" {
    availability_zones = ["us-west-2a"]
    name = "terraform-test-foobar5"
    max_size = 1
    min_size = 1
    health_check_grace_period = 300
    health_check_type = "ELB"
    force_delete = true
    termination_policies = ["OldestInstance"]
    launch_configuration = "${aws_launch_configuration.foobar.name}"
    tag {
        key = "Foo"
        value = "foo-bar"
        propagate_at_launch = true
    }
}

resource "aws_autoscaling_schedule" "foobar" {
    scheduled_action_name = "foobar"
    min_size = 0
    max_size = 1
    desired_capacity = 0
    recurrence = "0 8 * * *"
    autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
`)

var testAccAWSAutoscalingScheduleConfig_zeroValues = fmt.Sprintf(`
resource "aws_launch_configuration" "foobar" {
    name = "terraform-test-foobar5"
    image_id = "ami-21f78e11"
    instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "foobar" {
    availability_zones = ["us-west-2a"]
    name = "terraform-test-foobar5"
    max_size = 1
    min_size = 1
    health_check_grace_period = 300
    health_check_type = "ELB"
    force_delete = true
    termination_policies = ["OldestInstance"]
    launch_configuration = "${aws_launch_configuration.foobar.name}"
    tag {
        key = "Foo"
        value = "foo-bar"
        propagate_at_launch = true
    }
}

resource "aws_autoscaling_schedule" "foobar" {
    scheduled_action_name = "foobar"
    max_size = 0
    min_size = 0
    desired_capacity = 0
    start_time = "2018-01-16T07:00:00Z"
    end_time = "2018-01-16T13:00:00Z"
    autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
`)
