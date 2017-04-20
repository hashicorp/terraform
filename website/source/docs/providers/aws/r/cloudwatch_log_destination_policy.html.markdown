---
layout: "aws"
page_title: "AWS: aws_cloudwatch_log_destination_policy"
sidebar_current: "docs-aws-resource-cloudwatch-log-destination-policy"
description: |-
  Provides a CloudWatch Logs destination policy.
---

# aws\_cloudwatch\_log\_destination\_policy

Provides a CloudWatch Logs destination policy resource.

## Example Usage

```hcl
resource "aws_cloudwatch_log_destination" "test_destination" {
  name       = "test_destination"
  role_arn   = "${aws_iam_role.iam_for_cloudwatch.arn}"
  target_arn = "${aws_kinesis_stream.kinesis_for_cloudwatch.arn}"
}

data "aws_iam_policy_document" "test_destination_policy" {
  statement {
    effect = "Allow"

    principals = {
      type = "AWS"

      identifiers = [
        "123456789012",
      ]
    }

    actions = [
      "logs:PutSubscriptionFilter",
    ]

    resources = [
      "${aws_cloudwatch_log_destination.test_destination.arn}",
    ]
  }
}

resource "aws_cloudwatch_log_destination_policy" "test_destination_policy" {
  destination_name = "${aws_cloudwatch_log_destination.test_destination.name}"
  access_policy    = "${data.aws_iam_policy_document.test_destination_policy.json}"
}
```

## Argument Reference

The following arguments are supported:

* `destination_name` - (Required) A name for the subscription filter
* `access_policy` - (Required) The policy document. This is a JSON formatted string.

## Import

CloudWatch Logs destination policies can be imported using the `destination_name`, e.g.

```
$ terraform import aws_cloudwatch_log_destination_policy.test_destination_policy test_destination
```
