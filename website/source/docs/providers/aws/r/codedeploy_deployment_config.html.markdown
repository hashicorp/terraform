---
layout: "aws"
page_title: "AWS: aws_codedeploy_deployment_config"
sidebar_current: "docs-aws-resource-codedeploy-deployment-config"
description: |-
  Provides a CodeDeploy deployment config.
---

# aws\_codedeploy\_deployment\_config

Provides a CodeDeploy deployment config for an application

## Example Usage

```hcl
resource "aws_codedeploy_deployment_config" "foo" {
  deployment_config_name = "test-deployment-config"

  minimum_healthy_hosts {
    type  = "HOST_COUNT"
    value = 2
  }
}

resource "aws_codedeploy_deployment_group" "foo" {
  app_name               = "${aws_codedeploy_app.foo_app.name}"
  deployment_group_name  = "bar"
  service_role_arn       = "${aws_iam_role.foo_role.arn}"
  deployment_config_name = "${aws_codedeploy_deployment_config.foo.id}"

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

* `deployment_config_name` - (Required) The name of the deployment config.
* `minimum_healthy_hosts` - (Optional) A minimum_healthy_hosts block. Minimum Healthy Hosts are documented below.

A `minimum_healthy_hosts` block support the following:

* `type` - (Required) The type can either be `FLEET_PERCENT` or `HOST_COUNT`.
* `value` - (Required) The value when the type is `FLEET_PERCENT` represents the minimum number of healthy instances as
a percentage of the total number of instances in the deployment. If you specify FLEET_PERCENT, at the start of the
deployment, AWS CodeDeploy converts the percentage to the equivalent number of instance and rounds up fractional instances.
When the type is `HOST_COUNT`, the value represents the minimum number of healthy instances as an absolute value.

## Attributes Reference

The following attributes are exported:

* `id` - The deployment group's config name.
* `deployment_config_id` - The AWS Assigned deployment config id
