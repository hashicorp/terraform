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

```hcl
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
  app_name              = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name = "bar"
  service_role_arn      = "${aws_iam_role.foo_role.arn}"

  ec2_tag_filter {
    key   = "filterkey"
    type  = "KEY_AND_VALUE"
    value = "filtervalue"
  }

  trigger_configuration {
    trigger_events     = ["DeploymentFailure"]
    trigger_name       = "foo-trigger"
    trigger_target_arn = "foo-topic-arn"
  }

  auto_rollback_configuration {
    enabled = true
    events  = ["DEPLOYMENT_FAILURE"]
  }

  alarm_configuration {
    alarms  = ["my-alarm-name"]
    enabled = true
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
* `auto_rollback_configuration` - (Optional) The automatic rollback configuration associated with the deployment group, documented below.
* `alarm_configuration` - (Optional) A list of alarms associated with the deployment group, documented below.

Both ec2_tag_filter and on_premises_tag_filter blocks support the following:

* `key` - (Optional) The key of the tag filter.
* `type` - (Optional) The type of the tag filter, either KEY_ONLY, VALUE_ONLY, or KEY_AND_VALUE.
* `value` - (Optional) The value of the tag filter.

Add triggers to a Deployment Group to receive notifications about events related to deployments or instances in the group. Notifications are sent to subscribers of the SNS topic associated with the trigger. CodeDeploy must have permission to publish to the topic from this deployment group. Trigger Configurations support the following:

 * `trigger_events` - (Required) The event type or types for which notifications are triggered. Some values that are supported: `DeploymentStart`, `DeploymentSuccess`, `DeploymentFailure`, `InstanceStart`, `InstanceSuccess`, `InstanceFailure`.  See [the CodeDeploy documentation][1] for all possible values.
 * `trigger_name` - (Required) The name of the notification trigger.
 * `trigger_target_arn` - (Required) The ARN of the SNS topic through which notifications are sent.

You can configure a deployment group to automatically rollback when a deployment fails or when a monitoring threshold you specify is met. In this case, the last known good version of an application revision is deployed. Only one rollback configuration block is allowed.

 * `enabled` - (Optional) Indicates whether a defined automatic rollback configuration is currently enabled for this Deployment Group. If you enable automatic rollback, you must specify at least one event type.
 * `events` - (Optional) The event type or types that trigger a rollback. Supported types are `DEPLOYMENT_FAILURE` and `DEPLOYMENT_STOP_ON_ALARM`.

You can configure a deployment to stop when a CloudWatch alarm detects that a metric has fallen below or exceeded a defined threshold. Only one alarm configuration block is allowed.

 * `alarms` - (Optional) A list of alarms configured for the deployment group. A maximum of 10 alarms can be added to a deployment group.
 * `enabled` - (Optional) Indicates whether the alarm configuration is enabled. This option is useful when you want to temporarily deactivate alarm monitoring for a deployment group without having to add the same alarms again later.
 * `ignore_poll_alarm_failure` - (Optional) Indicates whether a deployment should continue if information about the current state of alarms cannot be retrieved from CloudWatch. The default value is `false`.
    * `true`: The deployment will proceed even if alarm status information can't be retrieved.
    * `false`: The deployment will stop if alarm status information can't be retrieved.

## Attributes Reference

The following attributes are exported:

* `id` - The deployment group's ID.
* `app_name` - The group's assigned application.
* `deployment_group_name` - The group's name.
* `service_role_arn` - The group's service role ARN.
* `autoscaling_groups` - The autoscaling groups associated with the deployment group.
* `deployment_config_name` - The name of the group's deployment config.

[1]: http://docs.aws.amazon.com/codedeploy/latest/userguide/monitoring-sns-event-notifications-create-trigger.html
