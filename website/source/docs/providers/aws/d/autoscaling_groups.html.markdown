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
data "aws_autoscaling_groups" "groups" {
  filter {
    name = "key"
    values = ["Team"]
  }

  filter {
    name = "value"
    values = ["Pets"]
  }
}

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

* `filter` - (Optional) A filter used to scope the list e.g. by tags. See [related docs](http://docs.aws.amazon.com/AutoScaling/latest/APIReference/API_Filter.html).
  * `name` - (Required) The name of the filter. The valid values are: `auto-scaling-group`, `key`, `value`, and `propagate-at-launch`.
  * `values` - (Required) The value of the filter.

## Attributes Reference

The following attributes are exported:

* `names` - A list of the Autoscaling Groups in the current region.
