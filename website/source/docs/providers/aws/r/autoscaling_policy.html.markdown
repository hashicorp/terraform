---
layout: "aws"
page_title: "AWS: aws_autoscaling_policy"
sidebar_current: "docs-aws-resource-autoscaling-policy"
description: |-
  Provides an AutoScaling Scaling Group resource.
---

# aws\_autoscaling\_policy

Provides an AutoScaling Scaling Policy resource.

~> **NOTE:** You may want to omit `desired_capacity` attribute from attached `aws_autoscaling_group`
when using autoscaling policies. It's good practice to pick either
[manual](https://docs.aws.amazon.com/AutoScaling/latest/DeveloperGuide/as-manual-scaling.html)
or [dynamic](https://docs.aws.amazon.com/AutoScaling/latest/DeveloperGuide/as-scale-based-on-demand.html)
(policy-based) scaling.

## Example Usage
```
resource "aws_autoscaling_policy" "bat" {
  name = "foobar3-terraform-test"
  scaling_adjustment = 4
  adjustment_type = "ChangeInCapacity"
  cooldown = 300
  autoscaling_group_name = "${aws_autoscaling_group.bar.name}"
}

resource "aws_autoscaling_group" "bar" {
  availability_zones = ["us-east-1a"]
  name = "foobar3-terraform-test"
  max_size = 5
  min_size = 2
  health_check_grace_period = 300
  health_check_type = "ELB"
  force_delete = true
  launch_configuration = "${aws_launch_configuration.foo.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the policy.
* `autoscaling_group_name` - (Required) The name or ARN of the group.
* `adjustment_type` - (Required) Specifies whether the adjustment is an absolute number or a percentage of the current capacity. Valid values are `ChangeInCapacity`, `ExactCapacity`, and `PercentChangeInCapacity`.
* `policy_type` - (Optional) The policy type, either "SimpleScaling" or "StepScaling". If this value isn't provided, AWS will default to "SimpleScaling."

The following arguments are only available to "SimpleScaling" type policies:

* `cooldown` - (Optional) The amount of time, in seconds, after a scaling activity completes and before the next scaling activity can start.
* `scaling_adjustment` - (Optional) The number of instances by which to scale. `adjustment_type` determines the interpretation of this number (e.g., as an absolute number or as a percentage of the existing Auto Scaling group size). A positive increment adds to the current capacity and a negative value removes from the current capacity.

The following arguments are only available to "StepScaling" type policies:

* `metric_aggregation_type` - (Optional) The aggregation type for the policy's metrics. Valid values are "Minimum", "Maximum", and "Average". Without a value, AWS will treat the aggregation type as "Average".
* `estimated_instance_warmup` - (Optional) The estimated time, in seconds, until a newly launched instance will contribute CloudWatch metrics. Without a value, AWS will default to the group's specified cooldown period.
* `step_adjustments` - (Optional) A set of adjustments that manage
group scaling. These have the following structure:

```
step_adjustment {
  scaling_adjustment = -1
  metric_interval_lower_bound = 1.0
  metric_interval_upper_bound = 2.0
}
step_adjustment {
  scaling_adjustment = 1
  metric_interval_lower_bound = 2.0
  metric_interval_upper_bound = 3.0
}
```

The following fields are available in step adjustments:

* `scaling_adjustment` - (Required) The number of members by which to
scale, when the adjustment bounds are breached. A positive value scales
up. A negative value scales down.
* `metric_interval_lower_bound` - (Optional) The lower bound for the
difference between the alarm threshold and the CloudWatch metric.
Without a value, AWS will treat this bound as infinity.
* `metric_interval_upper_bound` - (Optional) The upper bound for the
difference between the alarm threshold and the CloudWatch metric.
Without a value, AWS will treat this bound as infinity. The upper bound
must be greater than the lower bound.

The following arguments are supported for backwards compatibility but should not be used:

* `min_adjustment_step` - (Optional) Use `min_adjustment_magnitude` instead.

## Attribute Reference
* `arn` - The ARN assigned by AWS to the scaling policy.
* `name` - The scaling policy's name.
* `autoscaling_group_name` - The scaling policy's assigned autoscaling group.
* `adjustment_type` - The scaling policy's adjustment type.
* `policy_type` - The scaling policy's type.
