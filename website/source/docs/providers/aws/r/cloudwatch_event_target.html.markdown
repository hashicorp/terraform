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

```hcl
resource "aws_cloudwatch_event_target" "yada" {
  target_id = "Yada"
  rule      = "${aws_cloudwatch_event_rule.console.name}"
  arn       = "${aws_kinesis_stream.test_stream.arn}"
}

resource "aws_cloudwatch_event_rule" "console" {
  name        = "capture-ec2-scaling-events"
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
  name        = "terraform-kinesis-test"
  shard_count = 1
}
```

## Argument Reference

-> **Note:** `input` and `input_path` are mutually exclusive options.

-> **Note:** In order to be able to have your AWS Lambda function or
   SNS topic invoked by a CloudWatch Events rule, you must setup the right permissions
   using [`aws_lambda_permission`](https://www.terraform.io/docs/providers/aws/r/lambda_permission.html)
   or [`aws_sns_topic.policy`](https://www.terraform.io/docs/providers/aws/r/sns_topic.html#policy).
   More info [here](http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/EventsResourceBasedPermissions.html).

The following arguments are supported:

* `rule` - (Required) The name of the rule you want to add targets to.
* `target_id` - (Optional) The unique target assignment ID.  If missing, will generate a random, unique id.
* `arn` - (Required) The Amazon Resource Name (ARN) associated of the target.
* `input` - (Optional) Valid JSON text passed to the target.
* `input_path` - (Optional) The value of the [JSONPath](http://goessner.net/articles/JsonPath/)
	that is used for extracting part of the matched event when passing it to the target.
* `role_arn` - (Optional) The Amazon Resource Name (ARN) of the IAM role to be used for this target when the rule is triggered.
* `run_command_targets` - (Optional) Parameters used when you are using the rule to invoke Amazon EC2 Run Command. Documented below. A maximum of 5 are allowed.

`run_command_parameters` support the following:

* `key` - (Required) Can be either `tag:tag-key` or `InstanceIds`.
* `values` - (Required) If Key is `tag:tag-key`, Values is a list of tag values. If Key is `InstanceIds`, Values is a list of Amazon EC2 instance IDs.

