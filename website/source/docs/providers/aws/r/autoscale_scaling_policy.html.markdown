---
layout: "aws"
page_title: "AWS: aws_autoscaling_policy"
sidebar_current: "docs-aws-resource-autoscale-policy"
description: |-
  Provides an AutoScaling Scaling Group resource.
---

# aws\_autoscaling\_scaling\_policy

Provides an AutoScaling Scaling Policy resource.

## Example Usage
```
resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-east-1a"]
  name = "foobar3-terraform-test"
  max_size = 5
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  desired_capacity = 4
  force_delete = true
  launch_configuration = "${aws_launch_configuration.foobar.name}"

  tag {
    key = "foo"
    value = "bar"
    propagate_at_launch = true
  }
  tag {
    key = "lorem"
    value = "ipsum"
    propagate_at_launch = false
  }
}

resource "aws_autoscaling_policy" "bat" {
    name = "foobar3-terraform-test"
    scaling_adjustment = 4
    adjustment_type = "ChangeInCapacity"
    cooldown = 300
    autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
```

## Argument Reference

The following arguments are supported:

* `adjustment_type` - (Required) Specifies whether the `scaling_adjustment` is an absolute number or a percentage of the current capacity. Valid values are `ChangeInCapacity`, `ExactCapacity`, and `PercentChangeInCapacity`.
* `autoscaling_group_name` - (Required) The name or ARN of the group.
* `cooldown` - (Optional) The amount of time, in seconds, after a scaling activity completes and before the next scaling activity can start.
* `min_adjustment_step` - (Optional) Used with `adjustment_type` with the value `PercentChangeInCapacity`, the scaling policy changes the `desired_capacity` of the Auto Scaling group by at least the number of instances specified in the value.
* `name` - (Required) The name of the policy.
* `scaling_adjustment` - (Required) The number of instances by which to scale. `adjustment_type` determines the interpretation of this number (e.g., as an absolute number or as a percentage of the existing Auto Scaling group size). A positive increment adds to the current capacity and a negative value removes from the current capacity.

## Attribute Reference
* `policy_arn` - The ARN assigned by AWS to the scaling policy.
