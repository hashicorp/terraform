---
layout: "aws"
page_title: "AWS: aws_autoscaling_schedule"
sidebar_current: "docs-aws-resource-autoscaling-schedule"
description: |-
  Provides an AutoScaling Schedule resource.
---

# aws\_autoscaling\_schedule

Provides an AutoScaling Schedule resource.

## Example Usage

```hcl
resource "aws_autoscaling_group" "foobar" {
  availability_zones        = ["us-west-2a"]
  name                      = "terraform-test-foobar5"
  max_size                  = 1
  min_size                  = 1
  health_check_grace_period = 300
  health_check_type         = "ELB"
  force_delete              = true
  termination_policies      = ["OldestInstance"]
}

resource "aws_autoscaling_schedule" "foobar" {
  scheduled_action_name  = "foobar"
  min_size               = 0
  max_size               = 1
  desired_capacity       = 0
  start_time             = "2016-12-11T18:00:00Z"
  end_time               = "2016-12-12T06:00:00Z"
  autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
```

## Argument Reference

The following arguments are supported:

* `autoscaling_group_name` - (Required) The name or Amazon Resource Name (ARN) of the Auto Scaling group.
* `scheduled_action_name` - (Required) The name of this scaling action.
* `start_time` - (Optional) The time for this action to start, in "YYYY-MM-DDThh:mm:ssZ" format in UTC/GMT only (for example, 2014-06-01T00:00:00Z ).
                            If you try to schedule your action in the past, Auto Scaling returns an error message.
* `end_time` - (Optional) The time for this action to end, in "YYYY-MM-DDThh:mm:ssZ" format in UTC/GMT only (for example, 2014-06-01T00:00:00Z ).
                          If you try to schedule your action in the past, Auto Scaling returns an error message.
* `recurrence` - (Optional) The time when recurring future actions will start. Start time is specified by the user following the Unix cron syntax format.
* `min_size` - (Optional) The minimum size for the Auto Scaling group. Default
0.
* `max_size` - (Optional) The maximum size for the Auto Scaling group. Default
0.
* `desired_capacity` - (Optional) The number of EC2 instances that should be running in the group. Default 0.

~> **NOTE:** When `start_time` and `end_time` are specified with `recurrence` , they form the boundaries of when the recurring action will start and stop.

## Attribute Reference
* `arn` - The ARN assigned by AWS to the autoscaling schedule.
