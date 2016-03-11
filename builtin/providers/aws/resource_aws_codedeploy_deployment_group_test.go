package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCodeDeployDeploymentGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroupModified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_triggerConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_create,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group"),
				),
			},
		},
	})
}

func TestAccAWSCodeDeployDeploymentGroup_triggerConfiguration_multiple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeDeployDeploymentGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_createMultiple,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_updateMultiple,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeDeployDeploymentGroupExists("aws_codedeploy_deployment_group.foo_group"),
				),
			},
		},
	})
}

func TestValidateAWSCodeDeployTriggerEvent(t *testing.T) {
	validEvents := []string{
		"DeploymentStart",
		"DeploymentSuccess",
		"DeploymentFailure",
		"DeploymentStop",
		"InstanceStart",
		"InstanceSuccess",
		"InstanceFailure",
	}

	for _, v := range validEvents {
		_, errors := validateTriggerEvent(v, "trigger_event")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid trigger event type: %q", v, errors)
		}
	}

	invalidEvents := []string{
		"DeploymentStarts",
		"InstanceFail",
		"Foo",
		"",
	}

	for _, v := range invalidEvents {
		_, errors := validateTriggerEvent(v, "trigger_event")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid trigger event type: %q", v, errors)
		}
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

func testAccCheckAWSCodeDeployDeploymentGroupExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

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
		value = "filtervalue"
	}
}`

const baseCodeDeployConfig = `
resource "aws_codedeploy_app" "foo_app" {
  name = "foo"
}

resource "aws_iam_role_policy" "foo_policy" {
  name = "foo"
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
  name = "foo"
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
  name = "foo"
}

`

const testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_create = baseCodeDeployConfig + `
resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo"
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
  deployment_group_name = "foo"
  service_role_arn = "${aws_iam_role.foo_role.arn}"

  trigger_configuration {
    trigger_events = ["DeploymentSuccess", "DeploymentFailure"]
    trigger_name = "foo-trigger"
    trigger_target_arn = "${aws_sns_topic.foo_topic.arn}"
  }
}`

const testAccAWSCodeDeployDeploymentGroup_triggerConfiguration_createMultiple = baseCodeDeployConfig + `
resource "aws_sns_topic" "bar_topic" {
  name = "bar"
}

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo"
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
  name = "bar"
}

resource "aws_sns_topic" "baz_topic" {
  name = "baz"
}

resource "aws_codedeploy_deployment_group" "foo_group" {
  app_name = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "foo"
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
