---
layout: "aws"
page_title: "AWS: sns_topic"
sidebar_current: "docs-aws-resource-sns-topic"
description: |-
  Provides an SNS topic resource.
---

# aws\_sns\_topic

Provides an SNS topic resource

## Example Usage

```hcl
resource "aws_sns_topic" "user_updates" {
  name = "user-updates-topic"
}
```

## Message Delivery Status Arguments

The `<endpoint>_success_feedback_role_arn` and `<endpoint>_failure_feedback_role_arn` arguments are used to give Amazon SNS write access to use CloudWatch Logs on your behalf. The `<endpoint>_success_feedback_sample_rate` argument is for specifying the sample rate percentage (0-100) of successfully delivered messages. After you configure the  `<endpoint>_failure_feedback_role_arn` argument, then all failed message deliveries generate CloudWatch Logs.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The friendly name for the SNS topic
* `display_name` - (Optional) The display name for the SNS topic
* `policy` - (Optional) The fully-formed AWS policy as JSON
* `delivery_policy` - (Optional) The SNS delivery policy
* `application_success_feedback_role_arn` - (Optional) The IAM role permitted to receive success feedback for this topic
* `application_success_feedback_sample_rate` - (Optional) Percentage of success to sample
* `application_failure_feedback_role_arn` - (Optional) IAM role for failure feedback
* `http_success_feedback_role_arn` - (Optional) The IAM role permitted to receive success feedback for this topic
* `http_success_feedback_sample_rate` - (Optional) Percentage of success to sample
* `http_failure_feedback_role_arn` - (Optional) IAM role for failure feedback
* `lambda_success_feedback_role_arn` - (Optional) The IAM role permitted to receive success feedback for this topic
* `lambda_success_feedback_sample_rate` - (Optional) Percentage of success to sample
* `lambda_failure_feedback_role_arn` - (Optional) IAM role for failure feedback
* `sqs_success_feedback_role_arn` - (Optional) The IAM role permitted to receive success feedback for this topic
* `sqs_success_feedback_sample_rate` - (Optional) Percentage of success to sample
* `sqs_failure_feedback_role_arn` - (Optional) IAM role for failure feedback

## Attributes Reference

The following attributes are exported:

* `id` - The ARN of the SNS topic
* `arn` - The ARN of the SNS topic, as a more obvious property (clone of id)

## Import

SNS Topics can be imported using the `topic arn`, e.g.

```
$ terraform import aws_sns_topic.user_updates arn:aws:sns:us-west-2:0123456789012:my-topic
```