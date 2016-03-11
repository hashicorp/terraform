---
layout: "aws"
page_title: "AWS: aws_codedeploy_deployment_group"
sidebar_current: "docs-aws-resource-codedeploy-deployment-group"
description: |-
  Provides a CodeDeploy deployment group.
---

# aws\_codedeploy\_deployment\_group

Provides a CodeDeploy deployment group for an application

## Example Usage

```
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

    trigger_configuration {
        trigger_events = ["DeploymentFailure"]
        trigger_name = "foo-trigger"
        trigger_target_arn = "foo-topic-arn"
    }
}
```

## Argument Reference

The following arguments are supported:

* `app_name` - (Required) The name of the application.
* `deployment_group_name` - (Required) The name of the deployment group.
* `service_role_arn` - (Required) The service role ARN that allows deployments.
* `autoscaling_groups` - (Optional) Autoscaling groups associated with the deployment group.
* `deployment_config_name` - (Optional) The name of the group's deployment config. The default is "CodeDeployDefault.OneAtATime".
* `ec2_tag_filter` - (Optional) Tag filters associated with the group. See the AWS docs for details.
* `on_premises_instance_tag_filter` - (Optional) On premise tag filters associated with the group. See the AWS docs for details.
* `trigger_configuration` - (Optional) A Trigger Configuration block. Trigger Configurations are documented below.

Both ec2_tag_filter and on_premises_tag_filter blocks support the following:

* `key` - (Optional) The key of the tag filter.
* `type` - (Optional) The type of the tag filter, either KEY_ONLY, VALUE_ONLY, or KEY_AND_VALUE.
* `value` - (Optional) The value of the tag filter.

Add triggers to a Deployment Group to receive notifications about events related to deployments or instances in the group. Notifications are sent to subscribers of the SNS topic associated with the trigger. CodeDeploy must have permission to publish to the topic from this deployment group. Trigger Configurations support the following:

 * `trigger_events` - (Required) The event type or types for which notifications are triggered. The following values are supported: `DeploymentStart`, `DeploymentSuccess`, `DeploymentFailure`, `DeploymentStop`, `InstanceStart`, `InstanceSuccess`, `InstanceFailure`.
 * `trigger_name` - (Required) The name of the notification trigger.
 * `trigger_target_arn` - (Required) The ARN of the SNS topic through which notifications are sent.

## Attributes Reference

The following attributes are exported:

* `id` - The deployment group's ID.
* `app_name` - The group's assigned application.
* `deployment_group_name` - The group's name.
* `service_role_arn` - The group's service role ARN.
* `autoscaling_groups` - The autoscaling groups associated with the deployment group.
* `deployment_config_name` - The name of the group's deployment config.
