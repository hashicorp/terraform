---
layout: "aws"
page_title: "AWS: aws_autoscaling_group"
sidebar_current: "docs-aws-resource-autoscale"
---

# aws\_autoscaling\_group

Provides an AutoScaling Group resource.

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
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the auto scale group.
* `max_size` - (Required) The maximum size of the auto scale group.
* `min_size` - (Required) The minimum size of the auto scale group.
* `availability_zones` - (Required) A list of AZs to launch resources in.
* `launch_configuration` - (Required) The ID of the launch configuration to use.
* `health_check_grace_period` - (Optional) Time after instance comes into service before checking health.
* `health_check_type` - (Optional) "EC2" or "ELB". Controls how health checking is done.
* `desired_capacity` - (Optional) The number of Amazon EC2 instances that should be running in the group.
* `force_delete` - (Optional) Allows deleting the autoscaling group without waiting
   for all instances in the pool to terminate.
* `load_balancers` (Optional) A list of load balancer names to add to the autoscaling
   group names.
* `vpc_zone_identifier` (Optional) A list of vpc IDs to launch resources in.

## Attributes Reference

The following attributes are exported:

* `id` - The autoscaling group name.
* `availability_zones` - The availability zones of the autoscale group.
* `min_size` - The minimum size of the autoscale group
* `max_size` - The maximum size of the autoscale group
* `default_cooldown` - Time between a scaling activity and the succeeding scaling activity.
* `name` - The name of the autoscale group
* `health_check_grace_period` - Time after instance comes into service before checking health.
* `health_check_type` - "EC2" or "ELB". Controls how health checking is done.
* `desired_capacity` -The number of Amazon EC2 instances that should be running in the group.
* `launch_configuration` - The launch configuration of the autoscale group
* `vpc_zone_identifier` - The VPC zone identifier
* `load_balancers` (Optional) The load balancer names associated with the
   autoscaling group.
