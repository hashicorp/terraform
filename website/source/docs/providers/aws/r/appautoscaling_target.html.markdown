---
layout: "aws"
page_title: "AWS: aws_appautoscaling_target"
sidebar_current: "docs-aws-resource-appautoscaling-target"
description: |-
  Provides an Application AutoScaling ScalableTarget resource.
---

# aws\_appautoscaling\_target

Provides an Application AutoScaling ScalableTarget resource.

## Example Usage
```
resource "aws_appautoscaling_target" "tgt" {
  service_namespace = "ecs"
  resource_id = "service/clusterName/serviceName"
  scalable_dimension = "ecs:service:DesiredCount"
  role_arn = "${var.ecs_iam_role}"
  min_capacity = 1
  max_capacity = 4
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the policy.
* `resource_id` - (Required) The Resource ID on which you want the Application AutoScaling policy to apply to. For Amazon ECS services, this value is the resource type, followed by the cluster name and service name, such as `service/default/sample-webapp`.
* `scalable_dimension` - (Optional) The scalable dimension of the scalable target. The scalable dimension contains the  service  namespace,   resource  type,  and  scaling  property, such as `ecs:service:DesiredCount` for the desired task count of an Amazon ECS service. Defaults to `ecs:service:DesiredCount` since that is the only allowed value.
* `service_namespace` - (Optional) The AWS service namespace of the scalable target. Defaults to `ecs`, because that is currently the only supported option.
* `max_capacity` - (Required) The max capacity of the scalable target.
* `min_capacity` - (Required) The min capacity of the scalable target.
* `role_arn` - (Required) The ARN of the IAM role that allows Application AutoScaling to modify your scalable target on your behalf.


## Attribute Reference
* `arn` - The ARN assigned by AWS to the scaling policy.
* `name` - The scaling policy's name.
