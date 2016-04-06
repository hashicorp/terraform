---
layout: "aws"
page_title: "AWS: aws_autoscaling_notification"
sidebar_current: "docs-aws-resource-autoscaling-notification"
description: |-
  Provides an AutoScaling Group with Notification support
---

# aws\_autoscaling\_notification

Provides an AutoScaling Group with Notification support, via SNS Topics. Each of
the `notifications` map to a [Notification Configuration][2] inside Amazon Web
Services, and are applied to each AutoScaling Group you supply.

## Example Usage

Basic usage:

```
resource "aws_autoscaling_notification" "example_notifications" {
  group_names = [
    "${aws_autoscaling_group.bar.name}",
    "${aws_autoscaling_group.foo.name}",
  ]
  notifications  = [
    "autoscaling:EC2_INSTANCE_LAUNCH", 
    "autoscaling:EC2_INSTANCE_TERMINATE",
    "autoscaling:EC2_INSTANCE_LAUNCH_ERROR"
  ]
  topic_arn = "${aws_sns_topic.example.arn}"
}

resource "aws_sns_topic" "example" {
  name = "example-topic"
  # arn is an exported attribute
}

resource "aws_autoscaling_group" "bar" {
  name = "foobar1-terraform-test"
  [... ASG attributes ...]
}

resource "aws_autoscaling_group" "foo" {
  name = "barfoo-terraform-test"
  [... ASG attributes ...]
}
```

## Argument Reference

The following arguments are supported:

* `group_names` - (Required) A list of AutoScaling Group Names
* `notifications` - (Required) A list of Notification Types that trigger
notifications. Acceptable values are documented [in the AWS documentation here][1]
* `topic_arn` - (Required) The Topic ARN for notifications to be sent through

## Attributes Reference

The following attributes are exported:

* `group_names` 
* `notifications`
* `topic_arn` 


[1]: https://docs.aws.amazon.com/AutoScaling/latest/APIReference/API_NotificationConfiguration.html
[2]: https://docs.aws.amazon.com/AutoScaling/latest/APIReference/API_DescribeNotificationConfigurations.html 
