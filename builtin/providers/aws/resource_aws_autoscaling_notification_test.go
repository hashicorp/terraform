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

func TestAccAWSASGNotification_basic(t *testing.T) {
	var asgn autoscaling.DescribeNotificationConfigurationsOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckASGNDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccASGNotificationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckASGNotificationExists("aws_autoscaling_notification.example", []string{"foobar1-terraform-test"}, &asgn),
					testAccCheckAWSASGNotificationAttributes("aws_autoscaling_notification.example", &asgn),
				),
			},
		},
	})
}

func TestAccAWSASGNotification_update(t *testing.T) {
	var asgn autoscaling.DescribeNotificationConfigurationsOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckASGNDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccASGNotificationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckASGNotificationExists("aws_autoscaling_notification.example", []string{"foobar1-terraform-test"}, &asgn),
					testAccCheckAWSASGNotificationAttributes("aws_autoscaling_notification.example", &asgn),
				),
			},

			resource.TestStep{
				Config: testAccASGNotificationConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckASGNotificationExists("aws_autoscaling_notification.example", []string{"foobar1-terraform-test", "barfoo-terraform-test"}, &asgn),
					testAccCheckAWSASGNotificationAttributes("aws_autoscaling_notification.example", &asgn),
				),
			},
		},
	})
}

func TestAccAWSASGNotification_Pagination(t *testing.T) {
	var asgn autoscaling.DescribeNotificationConfigurationsOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckASGNDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccASGNotificationConfig_pagination,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckASGNotificationExists("aws_autoscaling_notification.example",
						[]string{
							"foobar3-terraform-test-0",
							"foobar3-terraform-test-1",
							"foobar3-terraform-test-2",
							"foobar3-terraform-test-3",
							"foobar3-terraform-test-4",
							"foobar3-terraform-test-5",
							"foobar3-terraform-test-6",
							"foobar3-terraform-test-7",
							"foobar3-terraform-test-8",
							"foobar3-terraform-test-9",
							"foobar3-terraform-test-10",
							"foobar3-terraform-test-11",
							"foobar3-terraform-test-12",
							"foobar3-terraform-test-13",
							"foobar3-terraform-test-14",
							"foobar3-terraform-test-15",
							"foobar3-terraform-test-16",
							"foobar3-terraform-test-17",
							"foobar3-terraform-test-18",
							"foobar3-terraform-test-19",
						}, &asgn),
					testAccCheckAWSASGNotificationAttributes("aws_autoscaling_notification.example", &asgn),
				),
			},
		},
	})
}

func testAccCheckASGNotificationExists(n string, groups []string, asgn *autoscaling.DescribeNotificationConfigurationsOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ASG Notification ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		opts := &autoscaling.DescribeNotificationConfigurationsInput{
			AutoScalingGroupNames: aws.StringSlice(groups),
			MaxRecords:            aws.Int64(100),
		}

		resp, err := conn.DescribeNotificationConfigurations(opts)
		if err != nil {
			return fmt.Errorf("Error describing notifications: %s", err)
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
			return fmt.Errorf("Error finding notification descriptions")
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

		// build a unique list of groups, notification types
		gRaw := make(map[string]bool)
		nRaw := make(map[string]bool)

		for _, n := range asgn.NotificationConfigurations {
			if *n.TopicARN == rs.Primary.Attributes["topic_arn"] {
				gRaw[*n.AutoScalingGroupName] = true
				nRaw[*n.NotificationType] = true
			}
		}

		// Grab the keys here as the list of Groups
		var gList []string
		for k, _ := range gRaw {
			gList = append(gList, k)
		}

		// Grab the keys here as the list of Types
		var nList []string
		for k, _ := range nRaw {
			nList = append(nList, k)
		}

		typeCount, _ := strconv.Atoi(rs.Primary.Attributes["notifications.#"])

		if len(nList) != typeCount {
			return fmt.Errorf("Error: Bad ASG Notification count, expected (%d), got (%d)", typeCount, len(nList))
		}

		groupCount, _ := strconv.Atoi(rs.Primary.Attributes["group_names.#"])

		if len(gList) != groupCount {
			return fmt.Errorf("Error: Bad ASG Group count, expected (%d), got (%d)", typeCount, len(gList))
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
		"autoscaling:EC2_INSTANCE_LAUNCH", 
		"autoscaling:EC2_INSTANCE_TERMINATE",
		"autoscaling:EC2_INSTANCE_LAUNCH_ERROR"
	]
	topic_arn = "${aws_sns_topic.topic_example.arn}"
}`

const testAccASGNotificationConfig_pagination = `
resource "aws_sns_topic" "user_updates" {
  name = "user-updates-topic"
}

resource "aws_launch_configuration" "foobar" {
  image_id = "ami-21f78e11"
  instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-west-2a"]
  count = 20
  name = "foobar3-terraform-test-${count.index}"
  max_size = 1
  min_size = 0
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 0
  force_delete = true
  termination_policies = ["OldestInstance"]
  launch_configuration = "${aws_launch_configuration.foobar.name}"
}

resource "aws_autoscaling_notification" "example" {
  group_names = [
    "${aws_autoscaling_group.bar.*.name}",
  ]
  notifications  = [
    "autoscaling:EC2_INSTANCE_LAUNCH",
    "autoscaling:EC2_INSTANCE_TERMINATE",
    "autoscaling:TEST_NOTIFICATION"
  ]
	topic_arn = "${aws_sns_topic.user_updates.arn}"
}`
