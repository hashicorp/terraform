---
layout: "aws"
page_title: "AWS: aws_appautoscaling_policy"
sidebar_current: "docs-aws-resource-appautoscaling-policy"
description: |-
  Provides an Application AutoScaling Policy resource.
---

# aws\_appautoscaling\_policy

Provides an Application AutoScaling Policy resource.

## Example Usage
```
resource "aws_appautoscaling_policy" "down" {
  name = "scale-down"
  service_namespace = "ecs"
  resource_id = "service/ecsclustername/servicename"
  scalable_dimension = "ecs:service:DesiredCount"

  adjustment_type = "ChangeInCapacity"
  cooldown = 60
  metric_aggregation_type = "Maximum"

  step_adjustment {
    metric_interval_lower_bound = 0
    scaling_adjustment = -1
  }
  depends_on = ["aws_appautoscaling_target.target"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the policy.
* `policy_type` - (Optional) Defaults to "StepScaling" because it is the only option available.
* `resource_id` - (Required) The Resource ID on which you want the Application AutoScaling policy to apply to. For Amazon ECS services, this value is the resource type, followed by the cluster name and service name, such as `service/default/sample-webapp`. For Amazon EC2 Spot fleet requests, the resource type is `spot-fleet-request`, and the identifier is the Spot fleet request ID; for example, `spot-fleet-request/sfr-73fbd2ce-aa30-494c-8788-1cee4EXAMPLE`.
* `scalable_dimension` - (Required) The scalable dimension of the scalable target. The scalable dimension contains the service namespace,   resource  type, and scaling property, such as `ecs:service:DesiredCount` for the desired task count of an Amazon ECS service, or `ec2:spot-fleet-request:TargetCapacity` for the target capacity of an Amazon EC2 Spot fleet request.
* `service_namespace` - (Required) The AWS service namespace of the scalable target. Valid values are `ecs` for Amazon ECS services and `ec2` Amazon EC2 Spot fleet requests.
* `adjustment_type` - (Required) Specifies whether the adjustment is an absolute number or a percentage of the current capacity. Valid values are `ChangeInCapacity`, `ExactCapacity`, and `PercentChangeInCapacity`.
* `cooldown` - (Required) The amount of time, in seconds, after a scaling activity completes and before the next scaling activity can start.
* `metric_aggregation_type` - (Required) The aggregation type for the policy's metrics. Valid values are "Minimum", "Maximum", and "Average". Without a value, AWS will treat the aggregation type as "Average".
* `step_adjustments` - (Optional) A set of adjustments that manage scaling. These have the following structure:
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

## Attribute Reference
* `arn` - The ARN assigned by AWS to the scaling policy.
* `name` - The scaling policy's name.
* `adjustment_type` - The scaling policy's adjustment type.
* `policy_type` - The scaling policy's type.
