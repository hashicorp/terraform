package aws

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeDeployDeploymentGroup_basic(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "app_name", "foo_app"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_group_name", "foo"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_config_name", "CodeDeployDefault.OneAtATime"),

					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2916377465.key", "filterkey"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2916377465.type", "KEY_AND_VALUE"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2916377465.value", "filtervalue"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroupModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "app_name", "foo_app"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_group_name", "bar"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "deployment_config_name", "CodeDeployDefault.OneAtATime"),

					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2369538975.key", "filterkey"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2369538975.type", "KEY_AND_VALUE"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo", "ec2_tag_filter.2369538975.value", "anotherfiltervalue"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_triggerConfiguration_basic(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_create,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group"),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
					}),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group"),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
						"DeploymentSuccess",
					}),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_triggerConfiguration_multiple(t *testing.T) {
	var group codedeploy.DeploymentGroupInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_createMultiple,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group"),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
					}),
					testAccCheckTriggerEvents(&group, "bar-trigger", []string{
						"InstanceFailure",
					}),
					testAccCheckTriggerTargetArn(&group, "bar-trigger",
						regexp.MustCompile("^arn:aws:sns:[^:]+:[0-9]{12}:bar-topic$")),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_updateMultiple,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group", &group),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "app_name", "foo-app"),
					resource.TestCheckResourceAttr(
						"aws_codedeploy_deployment_group.foo_group", "deployment_group_name", "foo-group"),
					testAccCheckTriggerEvents(&group, "foo-trigger", []string{
						"DeploymentFailure",
						"DeploymentStart",
						"DeploymentStop",
						"DeploymentSuccess",
					}),
					testAccCheckTriggerEvents(&group, "bar-trigger", []string{
						"InstanceFailure",
					}),
					testAccCheckTriggerTargetArn(&group, "bar-trigger",
						regexp.MustCompile("^arn:aws:sns:[^:]+:[0-9]{12}:baz-topic$")),
				),
			},
		},
	})
}

func TestValidateAWSCodeDeployTriggerEvent(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "DeploymentStart",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentStop",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentSuccess",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentFailure",
			ErrCount: 0,
		},
		{
			Value:    "InstanceStart",
			ErrCount: 0,
		},
		{
			Value:    "InstanceSuccess",
			ErrCount: 0,
		},
		{
			Value:    "InstanceFailure",
			ErrCount: 0,
		},
		{
			Value:    "DeploymentStarts",
			ErrCount: 1,
		},
		{
			Value:    "InstanceFail",
			ErrCount: 1,
		},
		{
			Value:    "Foo",
			ErrCount: 1,
		},
		{
			Value:    "",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateTriggerEvent(tc.Value, "trigger_event")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Trigger event validation failed for event type %q: %q", tc.Value, errors)
		}
	}
}

func TestBuildTriggerConfigs(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"trigger_events": schema.NewSet(schema.HashString, []interface{}{
				"DeploymentFailure",
			}),
			"trigger_name":       "foo-trigger",
			"trigger_target_arn": "arn:aws:sns:us-west-2:123456789012:foo-topic",
		},
	}

	expected := []*codedeploy.TriggerConfig{
		&codedeploy.TriggerConfig{
			TriggerEvents: []*string{
				aws.String("DeploymentFailure"),
			},
			TriggerName:      aws.String("foo-trigger"),
			TriggerTargetArn: aws.String("arn:aws:sns:us-west-2:123456789012:foo-topic"),
		},
	}

	actual := buildTriggerConfigs(input)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("buildTriggerConfigs output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func TestTriggerConfigsToMap(t *testing.T) {
	input := []*codedeploy.TriggerConfig{
		&codedeploy.TriggerConfig{
			TriggerEvents: []*string{
				aws.String("DeploymentFailure"),
				aws.String("InstanceFailure"),
			},
			TriggerName:      aws.String("bar-trigger"),
			TriggerTargetArn: aws.String("arn:aws:sns:us-west-2:123456789012:bar-topic"),
		},
	}

	expected := map[string]interface{}{
		"trigger_events": schema.NewSet(schema.HashString, []interface{}{
			"DeploymentFailure",
			"InstanceFailure",
		}),
		"trigger_name":       "bar-trigger",
		"trigger_target_arn": "arn:aws:sns:us-west-2:123456789012:bar-topic",
	}

	actual := triggerConfigsToMap(input)[0]

	fatal := false

	if actual["trigger_name"] != expected["trigger_name"] {
		fatal = true
	}

	if actual["trigger_target_arn"] != expected["trigger_target_arn"] {
		fatal = true
	}

	actualEvents := actual["trigger_events"].(*schema.Set)
	expectedEvents := expected["trigger_events"].(*schema.Set)
	if !actualEvents.Equal(expectedEvents) {
		fatal = true
	}

	if fatal {
		t.Fatalf("triggerConfigsToMap output is not correct.\nGot:\n%#v\nExpected:\n%#v\n",
			actual, expected)
	}
}

func testAccCheckTriggerEvents(group *codedeploy.DeploymentGroupInfo, triggerName string, expectedEvents []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		for _, actual := range group.TriggerConfigurations {
			if *actual.TriggerName == triggerName {

				numberOfEvents := len(actual.TriggerEvents)
				if numberOfEvents != len(expectedEvents) {
					return fmt.Errorf("Trigger events do not match. Expected: %d. Got: %d.",
						len(expectedEvents), numberOfEvents)
				}

				actualEvents := make([]string, 0, numberOfEvents)
				for _, event := range actual.TriggerEvents {
					actualEvents = append(actualEvents, *event)
				}
				sort.Strings(actualEvents)

				if !reflect.DeepEqual(actualEvents, expectedEvents) {
					return fmt.Errorf("Trigger events do not match.\nExpected: %v\nGot: %v\n",
						expectedEvents, actualEvents)
				}
				break
			}
		}
		return nil
	}
}

func testAccCheckTriggerTargetArn(group *codedeploy.DeploymentGroupInfo, triggerName string, r *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, actual := range group.TriggerConfigurations {
			if *actual.TriggerName == triggerName {
				if !r.MatchString(*actual.TriggerTargetArn) {
					return fmt.Errorf("Trigger target arn does not match regular expression.\nRegex: %v\nTriggerTargetArn: %v\n",
						r, *actual.TriggerTargetArn)
				}
				break
			}
		}
		return nil
	}
}

func testAccCheckAWSCodeDeployDeploymentGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).codedeployconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codedeploy_deployment_group" {
			continue
		}

		resp, err := conn.GetDeploymentGroup(&codedeploy.GetDeploymentGroupInput{
			ApplicationName:     aws.String(rs.Primary.Attributes["app_name"]),
			DeploymentGroupName: aws.String(rs.Primary.Attributes["deployment_group_name"]),
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "ApplicationDoesNotExistException" {
			continue
		}

		if err == nil {
			if resp.DeploymentGroupInfo.DeploymentGroupName != nil {
				return fmt.Errorf("CodeDeploy deployment group still exists:\n%#v", *resp.DeploymentGroupInfo.DeploymentGroupName)
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSCodeDeployDeploymentGroupExists(name string, group *codedeploy.DeploymentGroupInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).codedeployconn

		resp, err := conn.GetDeploymentGroup(&codedeploy.GetDeploymentGroupInput{
			ApplicationName:     aws.String(rs.Primary.Attributes["app_name"]),
			DeploymentGroupName: aws.String(rs.Primary.Attributes["deployment_group_name"]),
		})

		if err != nil {
			return err
		}

		*group = *resp.DeploymentGroupInfo

		return nil
	}
}

var testAccAWSCodeDeployDeploymentGroup = `
resource "aws_codedeploy_app" "foo_app" {
	name = "foo_app"
}

resource "aws_iam_role_policy" "foo_policy" {
	name = "foo_policy"
	role = "${aws_iam_role.foo_role.id}"
	policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"autoscaling:CompleteLifecycleAction",
				"autoscaling:DeleteLifecycleHook",
				"autoscaling:DescribeAutoScalingGroups",
				"autoscaling:DescribeLifecycleHooks",
				"autoscaling:PutLifecycleHook",
				"autoscaling:RecordLifecycleActionHeartbeat",
				"ec2:DescribeInstances",
				"ec2:DescribeInstanceStatus",
				"tag:GetTags",
				"tag:GetResources"
			],
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role" "foo_role" {
	name = "foo_role"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "",
			"Effect": "Allow",
			"Principal": {
				"Service": [
					"codedeploy.amazonaws.com"
				]
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}

resource "aws_codedeploy_deployment_group" "foo" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo"
	service_role_arn = "${aws_iam_role.foo_role.arn}"
	ec2_tag_filter {
		key = "filterkey"
		type = "KEY_AND_VALUE"
		value = "filtervalue"
	}
}`

var testAccAWSCodeDeployDeploymentGroupModified = `
resource "aws_codedeploy_app" "foo_app" {
	name = "foo_app"
}

resource "aws_iam_role_policy" "foo_policy" {
	name = "foo_policy"
	role = "${aws_iam_role.foo_role.id}"
	policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"autoscaling:CompleteLifecycleAction",
				"autoscaling:DeleteLifecycleHook",
				"autoscaling:DescribeAutoScalingGroups",
				"autoscaling:DescribeLifecycleHooks",
				"autoscaling:PutLifecycleHook",
				"autoscaling:RecordLifecycleActionHeartbeat",
				"ec2:DescribeInstances",
				"ec2:DescribeInstanceStatus",
				"tag:GetTags",
				"tag:GetResources"
			],
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role" "foo_role" {
	name = "foo_role"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "",
			"Effect": "Allow",
			"Principal": {
				"Service": [
					"codedeploy.amazonaws.com"
				]
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}

resource "aws_codedeploy_deployment_group" "foo" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "bar"
	service_role_arn = "${aws_iam_role.foo_role.arn}"
	ec2_tag_filter {
		key = "filterkey"
		type = "KEY_AND_VALUE"
		value = "anotherfiltervalue"
	}
}`

const baseCodeDeployConfig = `
resource "aws_codedeploy_app" "foo_app" {
	name = "foo-app"
}

resource "aws_iam_role_policy" "foo_policy" {
	name = "foo-policy"
	role = "${aws_iam_role.foo_role.id}"
	policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"autoscaling:CompleteLifecycleAction",
				"autoscaling:DeleteLifecycleHook",
				"autoscaling:DescribeAutoScalingGroups",
				"autoscaling:DescribeLifecycleHooks",
				"autoscaling:PutLifecycleHook",
				"autoscaling:RecordLifecycleActionHeartbeat",
				"ec2:DescribeInstances",
				"ec2:DescribeInstanceStatus",
				"tag:GetTags",
				"tag:GetResources",
				"sns:Publish"
			],
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role" "foo_role" {
	name = "foo-role"
	assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "",
			"Effect": "Allow",
			"Principal": {
				"Service": "codedeploy.amazonaws.com"
			},
			"Action": "sts:AssumeRole"
		}
	]
}
EOF
}

resource "aws_sns_topic" "foo_topic" {
	name = "foo-topic"
}

`

const testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_create = baseCodeDeployConfig + `
resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentFailure"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}
}`

const testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_update = baseCodeDeployConfig + `
resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentSuccess", "DeploymentFailure"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}
}`

const testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_createMultiple = baseCodeDeployConfig + `
resource "aws_sns_topic" "bar_topic" {
	name = "bar-topic"
}

resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentFailure"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}

	trigger_configuration {
		trigger_events = ["InstanceFailure"]
		trigger_name = "bar-trigger"
		trigger_target_arn = "${aws_sns_topic.bar_topic.arn}"
	}
}`

const testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_updateMultiple = baseCodeDeployConfig + `
resource "aws_sns_topic" "bar_topic" {
	name = "bar-topic"
}

resource "aws_sns_topic" "baz_topic" {
	name = "baz-topic"
}

resource "aws_codedeploy_deployment_group" "foo_group" {
	app_name = "${aws_codedeploy_app.foo_app.name}"
	deployment_group_name = "foo-group"
	service_role_arn = "${aws_iam_role.foo_role.arn}"

	trigger_configuration {
		trigger_events = ["DeploymentStart", "DeploymentSuccess", "DeploymentFailure", "DeploymentStop"]
		trigger_name = "foo-trigger"
		trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
	}

	trigger_configuration {
		trigger_events = ["InstanceFailure"]
		trigger_name = "bar-trigger"
		trigger_target_arn = "${aws_sns_topic.baz_topic.arn}"
	}
}`
