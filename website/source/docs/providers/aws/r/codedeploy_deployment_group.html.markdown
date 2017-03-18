---
layout: "aws"
page_title: "AWS: aws_codedeploy_deployment_group"
sidebar_current: "docs-aws-resource-codedeploy-deployment-group"
description: |-
  Provides a CodeDeploy deployment group.
---

# aws\_codedeploy\_deployment\_group

Provides a CodeDeploy Deployment Group for a CodeDeploy Application

## Example Usage

```hcl
resource "aws_iam_role" "example" {
  name = "example-role"

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

resource "aws_iam_role_policy" "example" {
  name = "example-policy"
  role = "${aws_iam_role.example.id}"

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
        "codedeploy:*",
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

resource "aws_codedeploy_app" "example" {
  name = "example-app"
}

resource "aws_sns_topic" "example" {
  name = "example-topic"
}

resource "aws_codedeploy_deployment_group" "example" {
  app_name              = "${aws_codedeploy_app.example.name}"
  deployment_group_name = "example-group"
  service_role_arn      = "${aws_iam_role.example.arn}"

  ec2_tag_filter {
    key   = "filterkey"
    type  = "KEY_AND_VALUE"
    value = "filtervalue"
  }

  trigger_configuration {
    trigger_events     = ["DeploymentFailure"]
    trigger_name       = "example-trigger"
    trigger_target_arn = "${aws_sns_topic.example.arn}"
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

### Using Blue Green Deployments

```hcl
resource "aws_codedeploy_app" "example" {
  name = "example-app"
}

resource "aws_codedeploy_deployment_group" "example" {
  app_name              = "${aws_codedeploy_app.example.name}"
  deployment_group_name = "example-group"
  service_role_arn      = "${aws_iam_role.example.arn}"

  deployment_style {
    deployment_option = "WITH_TRAFFIC_CONTROL"
    deployment_type   = "BLUE_GREEN"
  }

  load_balancer_info {
    elb_info {
      name = "example-elb"
    }
  }

  blue_green_deployment_config {
    deployment_ready_option {
      action_on_timeout    = "STOP_DEPLOYMENT"
      wait_time_in_minutes = 60
    }

    green_fleet_provisioning_option {
      action = "DISCOVER_EXISTING"
    }

    terminate_blue_instances_on_deployment_success {
      action = "KEEP_ALIVE"
    }
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
* `trigger_configuration` - (Optional) A Trigger Configuration block (documented below).
* `auto_rollback_configuration` - (Optional) The automatic rollback configuration associated with the deployment group (documented below).
* `alarm_configuration` - (Optional) Information about alarms associated with the deployment group (documented below).
* `deployment_style` - (Optional) Information about the type of deployment, either standard or blue/green, you want to run and whether to route deployment traffic behind a load balancer (documented below).
* `load_balancer_info` - (Optional) Information about the load balancer to use in a blue/green deployment (documented below).
* `blue_green_deployment_config` - (Optional) Information about blue/green deployment options for a deployment group (documented below).

### Tag Filters
Both `ec2_tag_filter` and `on_premises_tag_filter` support the following:

* `key` - (Optional) The key of the tag filter.
* `type` - (Optional) The type of the tag filter, either `KEY_ONLY`, `VALUE_ONLY`, or `KEY_AND_VALUE`.
* `value` - (Optional) The value of the tag filter.

### Trigger Configuration
Add triggers to a Deployment Group to receive notifications about events related to deployments or instances in the group. Notifications are sent to subscribers of the **SNS** topic associated with the trigger. _CodeDeploy must have permission to publish to the topic from this deployment group_. `trigger_configuration` supports the following:

 * `trigger_events` - (Required) The event type or types for which notifications are triggered. Some values that are supported: `DeploymentStart`, `DeploymentSuccess`, `DeploymentFailure`, `InstanceStart`, `InstanceSuccess`, `InstanceFailure`.  See [the CodeDeploy documentation][1] for all possible values.
 * `trigger_name` - (Required) The name of the notification trigger.
 * `trigger_target_arn` - (Required) The ARN of the SNS topic through which notifications are sent.

### Auto Rollback Configuration
You can configure a deployment group to automatically rollback when a deployment fails or when a monitoring threshold you specify is met. In this case, the last known good version of an application revision is deployed. `auto_rollback_configuration` supports the following:

 * `enabled` - (Optional) Indicates whether a defined automatic rollback configuration is currently enabled for this Deployment Group. If you enable automatic rollback, you must specify at least one event type.
 * `events` - (Optional) The event type or types that trigger a rollback. Supported types are `DEPLOYMENT_FAILURE` and `DEPLOYMENT_STOP_ON_ALARM`.

_Only one `auto_rollback_ configuration` block is allowed_.

### Alarm Configuration
You can configure a deployment to stop when a **CloudWatch** alarm detects that a metric has fallen below or exceeded a defined threshold. `alarm_configuration` supports the following:

 * `alarms` - (Optional) A list of alarms configured for the deployment group. _A maximum of 10 alarms can be added to a deployment group_.
 * `enabled` - (Optional) Indicates whether the alarm configuration is enabled. This option is useful when you want to temporarily deactivate alarm monitoring for a deployment group without having to add the same alarms again later.
 * `ignore_poll_alarm_failure` - (Optional) Indicates whether a deployment should continue if information about the current state of alarms cannot be retrieved from CloudWatch. The default value is `false`.
    * `true`: The deployment will proceed even if alarm status information can't be retrieved.
    * `false`: The deployment will stop if alarm status information can't be retrieved.

_Only one `alarm_configuration` block is allowed_.

### Deployment Style
You can configure the type of deployment, either standard or blue/green, you want to run and whether to route deployment traffic behind a load balancer. `deployment_style` supports the following:

* `deployment_option` - (Optional) Indicates whether to route deployment traffic behind a load balancer. Valid Values are `WITH_TRAFFIC_CONTROL` or `WITHOUT_TRAFFIC_CONTROL`.

* `deployment_type` - (Optional) Indicates whether to run a standard deployment or a blue/green deployment. Valid Values are `IN_PLACE` or `BLUE_GREEN`.

  * `IN_PLACE` deployment type is not supported with the `WITH_TRAFFIC_CONTROL` deployment option.
  * `BLUE_GREEN` deployment type is not supported with the `WITHOUT_TRAFFIC_CONTROL` deployment option.

_Only one `deployment_style` block is allowed_.

### Load Balancer Info
You can configure the **Elastic Load Balancer** to use in a blue/green deployment. `load_balancer_info` supports the following:

* `elb_info_list` - (Optional) The load balancers to use in a blue/green deployment.

_Only one `load_balancer_info` block is allowed_.

`elb_info_list` is a list of `elb_info`. `elb_info` supports the following:

* `name` - (Optional) The name of the load balancer that will be used to route traffic from original instances to replacement instances in a blue/green deployment.

_Only one `elb_info` block is supported at this time._

### Blue Green Deployment Configuration
You can configure options for a blue/green deployment. `blue_green_deployment_config` supports the following:

* `deployment_ready_option` - (Optional) Information about the action to take when newly provisioned instances are ready to receive traffic in a blue/green deployment (documented below).

* `green_fleet_provisioning_option` - (Optional) Information about how instances are provisioned for a replacement environment in a blue/green deployment (documented below).

* `terminate_blue_instances_on_deployment_success` - (Optional) Information about whether to terminate instances in the original fleet during a blue/green deployment (documented below).

_Only one `blue_green_deployment_config` block is allowed_.

You can configure how traffic is rerouted to instances in a replacement environment in a blue/green deployment. `deployment_ready_option` supports the following:

* `action_on_timeout` - (Optional) When to reroute traffic from an original environment to a replacement environment in a blue/green deployment.

  * `CONTINUE_DEPLOYMENT`: Register new instances with the load balancer immediately after the new application revision is installed on the instances in the replacement environment.
  * `STOP_DEPLOYMENT`: Do not register new instances with load balancer unless traffic is rerouted manually. If traffic is not rerouted manually before the end of the specified wait period, the deployment status is changed to Stopped.

* `wait_time_in_minutes` - (Optional) The number of minutes to wait before the status of a blue/green deployment changed to Stopped if rerouting is not started manually. Applies only to the `STOP_DEPLOYMENT` option for `action_on_timeout`.

You can configure how instances will be added to the replacement environment in a blue/green deployment. `green_fleet_provisioning_option` supports the following:

* `action` - (Optional) The method used to add instances to a replacement environment.

  * `DISCOVER_EXISTING`: Use instances that already exist or will be created manually.
  * `COPY_AUTO_SCALING_GROUP`: Use settings from a specified **Auto Scaling** group to define and create instances in a new Auto Scaling group. _Exactly one Auto Scaling group must be specifed_ when selecting `COPY_AUTO_SCALING_GROUP`. Use `autoscaling_groups` to specify the Auto Scaling group.

You can configure how instances in the original environment are terminated when a blue/green deployment is successful. `terminate_blue_instances_on_deployment_success` supports the following:

* `action` - (Optional) The action to take on instances in the original environment after a successful blue/green deployment.

  * `TERMINATE`: Instances are terminated after a specified wait time.

  * `KEEP_ALIVE`: Instances are left running after they are deregistered from the load balancer and removed from the deployment group.

* `termination_wait_time_in_minutes` - (Optional) The number of minutes to wait after a successful blue/green deployment before terminating instances from the original environment.

## Attributes Reference

The following attributes are exported:

* `id` - The deployment group's ID.
* `app_name` - The group's assigned application.
* `deployment_group_name` - The group's name.
* `service_role_arn` - The group's service role ARN.
* `autoscaling_groups` - The autoscaling groups associated with the deployment group.
* `deployment_config_name` - The name of the group's deployment config.

[1]: http://docs.aws.amazon.com/codedeploy/latest/userguide/monitoring-sns-event-notifications-create-trigger.html
