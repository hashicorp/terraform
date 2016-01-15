---
layout: "aws"
page_title: "AWS: aws_autoscaling_metrics_collection"
sidebar_current: "docs-aws-resource-autoscaling-metrics-collection"
description: |-
  Enables / Disables Autoscaling Group Metrics Collection.
---

# aws\_autoscaling\_metrics\_collection

Enables / Disables Autoscaling Group Metrics Collection.

~> **NOTE:** You can only enable metrics collection for an Autoscaling Group if `enable_monitoring` 
in it's underlying launch configuration for the group is set to `True`.

## Example Usage

```
resource "aws_launch_configuration" "foobar" {
    name = "web_config"
    image_id = "ami-408c7f28"
    instance_type = "t1.micro"
    enable_monitoring = true
}

resource "aws_autoscaling_group" "foobar" {
    availability_zones = ["us-west-2a"]
    name = "terraform-test-foobar5"
    health_check_type = "EC2"
    termination_policies = ["OldestInstance"]
    launch_configuration = "${aws_launch_configuration.foobar.name}"
    tag {
        key = "Foo"
        value = "foo-bar"
        propagate_at_launch = true
    }
}

resource "aws_autoscaling_metrics_collection" "test" {
  autoscaling_group_name = "${aws_autoscaling_group.bar.name}"
  granularity = "1Minute"
  metrics = ["GroupTotalInstances",
  	     "GroupPendingInstances",
  	     "GroupTerminatingInstances",
  	     "GroupDesiredCapacity",
  	     "GroupMaxSize"
  ]
}
```

## Argument Reference

The following arguments are supported:

* `autoscaling_group_name` - (Required) The name of the Auto Scaling group to which you want to enable / disable metrics collection
* `granularity` - (Required) The granularity to associate with the metrics to collect. The only valid value is `1Minute`.
* `metrics` - (Required) A list of metrics to collect. The allowed values are `GroupMinSize`, `GroupMaxSize`, `GroupDesiredCapacity`, `GroupInServiceInstances`, `GroupPendingInstances`, `GroupStandbyInstances`, `GroupTerminatingInstances`, `GroupTotalInstances`.