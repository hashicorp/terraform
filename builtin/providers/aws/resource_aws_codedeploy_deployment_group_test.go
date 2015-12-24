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
