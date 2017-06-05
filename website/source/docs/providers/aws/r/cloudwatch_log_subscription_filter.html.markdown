---
layout: "aws"
page_title: "AWS: aws_cloudwatch_log_subscription_filter"
sidebar_current: "docs-aws-resource-cloudwatch-log-subscription-filter"
description: |-
  Provides a CloudWatch Logs subscription filter.
---

# aws\_cloudwatch\_log\_subscription\_filter

Provides a CloudWatch Logs subscription filter resource.

## Example Usage

```hcl
resource "aws_cloudwatch_log_subscription_filter" "test_lambdafunction_logfilter" {
  name            = "test_lambdafunction_logfilter"
  role_arn        = "${aws_iam_role.iam_for_lambda.arn}"
  log_group_name  = "/aws/lambda/example_lambda_name"
  filter_pattern  = "logtype test"
  destination_arn = "${aws_kinesis_stream.test_logstream.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name for the subscription filter
* `destination_arn` - (Required) The ARN of the destination to deliver matching log events to. Kinesis stream or Lambda function ARN.
* `filter_pattern` - (Required) A valid CloudWatch Logs filter pattern for subscribing to a filtered stream of log events.
* `log_group_name` - (Required) The name of the log group to associate the subscription filter with
* `role_arn` - (Optional) The ARN of an IAM role that grants Amazon CloudWatch Logs permissions to deliver ingested log events to the destination. If you use Lambda as a destination, you should skip this argument and use `aws_lambda_permission` resource for granting access from CloudWatch logs to the destination Lambda function. 

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) specifying the log subscription filter.
