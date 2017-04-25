---
layout: "aws"
page_title: "AWS: aws_cloudwatch_event_rule"
sidebar_current: "docs-aws-resource-cloudwatch-event-rule"
description: |-
  Provides a CloudWatch Event Rule resource.
---

# aws\_cloudwatch\_event\_rule

Provides a CloudWatch Event Rule resource.

## Example Usage

```hcl
resource "aws_cloudwatch_event_rule" "console" {
  name        = "capture-aws-sign-in"
  description = "Capture each AWS Console Sign In"

  event_pattern = <<PATTERN
{
  "detail-type": [
    "AWS Console Sign In via CloudTrail"
  ]
}
PATTERN
}

resource "aws_cloudwatch_event_target" "sns" {
  rule      = "${aws_cloudwatch_event_rule.console.name}"
  target_id = "SendToSNS"
  arn       = "${aws_sns_topic.aws_logins.arn}"
}

resource "aws_sns_topic" "aws_logins" {
  name = "aws-console-logins"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The rule's name.
* `schedule_expression` - (Required, if `event_pattern` isn't specified) The scheduling expression.
	For example, `cron(0 20 * * ? *)` or `rate(5 minutes)`.
* `event_pattern` - (Required, if `schedule_expression` isn't specified) Event pattern
	described a JSON object.
	See full documentation of [CloudWatch Events and Event Patterns](http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/CloudWatchEventsandEventPatterns.html) for details.
* `description` - (Optional) The description of the rule.
* `role_arn` - (Optional) The Amazon Resource Name (ARN) associated with the role that is used for target invocation.
* `is_enabled` - (Optional) Whether the rule should be enabled (defaults to `true`).

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) of the rule.


## Import

Cloudwatch Event Rules can be imported using the `name`, e.g.

```
$ terraform import aws_cloudwatch_event_rule.console capture-console-sign-in
```