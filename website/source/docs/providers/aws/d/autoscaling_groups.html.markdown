---
layout: "aws"
page_title: "AWS: aws_autoscaling_groups"
sidebar_current: "docs-aws-datasource-autoscaling-groups"
description: |-
    Provides a list of Autoscaling Groups within a specific region.
---

# aws\_autoscaling\_groups

The Autoscaling Groups data source allows access to the list of AWS
ASGs within a specific region. This will allow you to pass a list of AutoScaling Groups to other resources.

## Example Usage

```hcl
data "aws_autoscaling_groups" "groups" {}

resource "aws_autoscaling_notification" "slack_notifications" {
  group_names = ["${data.aws_autoscaling_groups.groups.names}"]

  notifications = [
    "autoscaling:EC2_INSTANCE_LAUNCH",
    "autoscaling:EC2_INSTANCE_TERMINATE",
    "autoscaling:EC2_INSTANCE_LAUNCH_ERROR",
    "autoscaling:EC2_INSTANCE_TERMINATE_ERROR",
  ]

  topic_arn = "TOPIC ARN"
}
```

## Argument Reference

The data source currently takes no arguments as it uses the current region in which the provider is currently operating.

## Attributes Reference

The following attributes are exported:

* `names` - A list of the Autoscaling Groups in the current region.
