package aws

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestBuildNotificationSlice(t *testing.T) {
	a := "autoscaling:one"
	b := "autoscaling:two"

	cases := []struct {
		Input  []string
		Output []*string
	}{
		{[]string{"one", "two"}, []*string{&a, &b}},
		{[]string{"autoscaling:one", "two"}, []*string{&a, &b}},
	}

	for _, tc := range cases {
		actual := buildNotificationTypesSlice(tc.Input)
		for i, a := range actual {
			if *tc.Output[i] != *a {
				t.Fatalf("bad converstion:\n\tinput: %s\n\toutput: %s", *tc.Output[i], *a)
			}
		}
	}
}

func TestAccASGNotification_basic(t *testing.T) {
	var asgn autoscaling.DescribeNotificationConfigurationsOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckASGNDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccASGNotificationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckASGNotificationExists("aws_autoscaling_notification.example", &asgn),
					testAccCheckAWSASGNotificationAttributes("aws_autoscaling_notification.example", &asgn),
				),
			},
		},
	})
}

func TestAccASGNotification_update(t *testing.T) {
	var asgn autoscaling.DescribeNotificationConfigurationsOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckASGNDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccASGNotificationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckASGNotificationExists("aws_autoscaling_notification.example", &asgn),
					testAccCheckAWSASGNotificationAttributes("aws_autoscaling_notification.example", &asgn),
				),
			},

			resource.TestStep{
				Config: testAccASGNotificationConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckASGNotificationExists("aws_autoscaling_notification.example", &asgn),
					testAccCheckAWSASGNotificationAttributes("aws_autoscaling_notification.example", &asgn),
				),
			},
		},
	})
}

func testAccCheckASGNotificationExists(n string, asgn *autoscaling.DescribeNotificationConfigurationsOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ASG Notification ID is set")
		}

		// var groups []*string
		// groupCount, _ := strconv.Atoi(rs.Primary.Attributes["group_names.#"])
		// for i := 0; i < groupCount; i++ {
		// 	key := fmt.Sprintf("group_names.%d", i)
		// 	groups = append(groups, aws.String(rs.Primary.Attributes[key]))
		// }
		groups := []*string{aws.String("foobar1-terraform-test")}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		opts := &autoscaling.DescribeNotificationConfigurationsInput{
			AutoScalingGroupNames: groups,
		}

		resp, err := conn.DescribeNotificationConfigurations(opts)
		if err != nil {
			return fmt.Errorf("Error describing notifications")
		}

		*asgn = *resp

		return nil
	}
}

func testAccCheckASGNDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscaling_notification" {
			continue
		}

		groups := []*string{aws.String("foobar1-terraform-test")}
		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		opts := &autoscaling.DescribeNotificationConfigurationsInput{
			AutoScalingGroupNames: groups,
		}

		resp, err := conn.DescribeNotificationConfigurations(opts)
		if err != nil {
			return fmt.Errorf("Error describing notifications")
		}

		if len(resp.NotificationConfigurations) != 0 {
			fmt.Errorf("Error finding notification descriptions")
		}

	}
	return nil
}

func testAccCheckAWSASGNotificationAttributes(n string, asgn *autoscaling.DescribeNotificationConfigurationsOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ASG Notification ID is set")
		}

		if len(asgn.NotificationConfigurations) == 0 {
			return fmt.Errorf("Error: no ASG Notifications found")
		}

		var notifications []*autoscaling.NotificationConfiguration
		for _, n := range asgn.NotificationConfigurations {
			if *n.TopicARN == rs.Primary.Attributes["topic_arn"] {
				notifications = append(notifications, n)
			}
		}

		typeCount, _ := strconv.Atoi(rs.Primary.Attributes["notifications.#"])

		if len(notifications) != typeCount {
			return fmt.Errorf("Error: Bad ASG Notification count, expected (%d), got (%d)", typeCount, len(notifications))
		}

		return nil
	}
}

const testAccASGNotificationConfig_basic = `
resource "aws_sns_topic" "topic_example" {
  name = "user-updates-topic"
}

resource "aws_launch_configuration" "foobar" {
  name = "foobarautoscaling-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar1-terraform-test"
  max_size = 1
  min_size = 1
  health_check_grace_period = 100
  health_check_type = "ELB"
  desired_capacity = 1
  force_delete = true
  termination_policies = ["OldestInstance"]
  launch_configuration = "${aws_launch_configuration.foobar.name}"
}

resource "aws_autoscaling_notification" "example" {
  group_names     = ["${aws_autoscaling_group.bar.name}"]
  notifications  = [
	"autoscaling:EC2_INSTANCE_LAUNCH", 
	"autoscaling:EC2_INSTANCE_TERMINATE", 
  ]
  topic_arn = "${aws_sns_topic.topic_example.arn}"
}
`

const testAccASGNotificationConfig_update = `
resource "aws_sns_topic" "user_updates" {
  name = "user-updates-topic"
}

resource "aws_launch_configuration" "foobar" {
  name = "foobarautoscaling-terraform-test"
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  name = "foobar1-terraform-test"
  max_size = 1
  min_size = 1
  health_check_grace_period = 100
  health_check_type = "ELB"
  desired_capacity = 1
  force_delete = true
  termination_policies = ["OldestInstance"]
  launch_configuration = "${aws_launch_configuration.foobar.name}"
}

resource "aws_autoscaling_group" "foo" {
  availability_zones = ["us-west-2b"]
  name = "barfoo-terraform-test"
  max_size = 1
  min_size = 1
  health_check_grace_period = 200
  health_check_type = "ELB"
  desired_capacity = 1
  force_delete = true
  termination_policies = ["OldestInstance"]
  launch_configuration = "${aws_launch_configuration.foobar.name}"
}

resource "aws_autoscaling_notification" "example" {
  group_names     = [
	"${aws_autoscaling_group.bar.name}",
	"${aws_autoscaling_group.foo.name}",
	]
  notifications  = [
    "EC2_INSTANCE_LAUNCH", 
    "EC2_INSTANCE_TERMINATE",
    "EC2_INSTANCE_LAUNCH_ERROR"
  ]
  topic_arn = "${aws_sns_topic.user_updates.arn}"
}`
