---
layout: "aws"
page_title: "AWS: aws_cloudwatch_event_target"
sidebar_current: "docs-aws-resource-cloudwatch-event-target"
description: |-
  Provides a CloudWatch Event Target resource.
---

# aws\_cloudwatch\_event\_target

Provides a CloudWatch Event Target resource.

## Example Usage

```
resource "aws_cloudwatch_event_target" "yada" {
  target_id = "Yada"
  rule = "${aws_cloudwatch_event_rule.console.name}"
  arn = "${aws_kinesis_stream.test_stream.arn}"
}

resource "aws_cloudwatch_event_rule" "console" {
  name = "capture-ec2-scaling-events"
  description = "Capture all EC2 scaling events"
  event_pattern = <<PATTERN
{
  "source": [
    "aws.autoscaling"
  ],
  "detail-type": [
    "EC2 Instance Launch Successful",
    "EC2 Instance Terminate Successful",
    "EC2 Instance Launch Unsuccessful",
    "EC2 Instance Terminate Unsuccessful"
  ]
}
PATTERN
}

resource "aws_kinesis_stream" "test_stream" {
    name = "terraform-kinesis-test"
    shard_count = 1
}
```

## Argument Reference

-> **Note:** `input` and `input_path` are mutually exclusive options.

The following arguments are supported:

* `rule` - (Required) The name of the rule you want to add targets to.
* `target_id` - (Required) The unique target assignment ID.
* `arn` - (Required) The Amazon Resource Name (ARN) associated of the target.
* `input` - (Optional) Valid JSON text passed to the target.
* `input_path` - (Optional) The value of the [JSONPath](http://goessner.net/articles/JsonPath/)
	that is used for extracting part of the matched event when passing it to the target.
