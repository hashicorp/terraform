---
layout: "aws"
page_title: "AWS: sns_topic_subscription"
sidebar_current: "docs-aws-resource-sns-topic-subscription"
description: |-
  Provides a resource for subscribing to SNS topics.
---

# aws\_sns\_topic\_subscription

  Provides a resource for subscribing to SNS topics. Requires that an SNS topic exist for the subscription to attach to.
This resource allows you to automatically place messages sent to SNS topics in SQS queues, send them as HTTP(S) POST requests
to a given endpoint, send SMS messages, or notify devices / applications. The most likely use case for Terraform users will
probably be SQS queues.

~> **NOTE:** If SNS topic and SQS queue are in different AWS regions it is important to place the "aws_sns_topic_subscription" into the terraform configuration of the region with the SQS queue. If "aws_sns_topic_subscription" is placed in the terraform configuration of the region with the SNS topic terraform will fail to create the subscription.

~> **NOTE:** Setup of cross-account subscriptions from SNS topics to SQS queues requires Terraform to have access to BOTH accounts.

~> **NOTE:** If SNS topic and SQS queue are in different AWS accounts but the same region it is important to place the "aws_sns_topic_subscription" into the terraform configuration of the account with the SQS queue. If "aws_sns_topic_subscription" is placed in the terraform configuration of the account with the SNS topic terraform creates the subscriptions but does not keep state and tries to re-create the subscription at every apply.

~> **NOTE:** If SNS topic and SQS queue are in different AWS accounts and different AWS regions it is important to recognize that the subscription needs to be initiated from the account with the SQS queue but in the region of the SNS topic.

## Example Usage

You can directly supply a topic and ARN by hand in the `topic_arn` property along with the queue ARN:

```hcl
resource "aws_sns_topic_subscription" "user_updates_sqs_target" {
  topic_arn = "arn:aws:sns:us-west-2:432981146916:user-updates-topic"
  protocol  = "sqs"
  endpoint  = "arn:aws:sqs:us-west-2:432981146916:terraform-queue-too"
}
```

Alternatively you can use the ARN properties of a managed SNS topic and SQS queue:

```hcl
resource "aws_sns_topic" "user_updates" {
  name = "user-updates-topic"
}

resource "aws_sqs_queue" "user_updates_queue" {
  name = "user-updates-queue"
}

resource "aws_sns_topic_subscription" "user_updates_sqs_target" {
  topic_arn = "${aws_sns_topic.user_updates.arn}"
  protocol  = "sqs"
  endpoint  = "${aws_sqs_queue.user_updates_queue.arn}"
}
```

You can subscribe SNS topics to SQS queues in different Amazon accounts and regions:

```hcl
/*
#
# Variables
#
*/
variable "sns" {
  default = {
    account-id    = "111111111111"
    role-name     = "service/service-hashicorp-terraform"
    name          = "example-sns-topic"
    display_name  = "example"
    region        = "us-west-1"
  }
}

variable "sqs" {
  default = {
    account-id    = "222222222222"
    role-name     = "service/service-hashicorp-terraform"
    name          = "example-sqs-queue"
    region        = "us-east-1"
  }
}

data "aws_iam_policy_document" "sns-topic-policy" {
  policy_id = "__default_policy_ID"

  statement {
    actions = [
      "SNS:Subscribe",
      "SNS:SetTopicAttributes",
      "SNS:RemovePermission",
      "SNS:Receive",
      "SNS:Publish",
      "SNS:ListSubscriptionsByTopic",
      "SNS:GetTopicAttributes",
      "SNS:DeleteTopic",
      "SNS:AddPermission",
    ]

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceOwner"

      values = [
        "${var.sns["account-id"]}",
      ]
    }

    effect = "Allow"

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    resources = [
      "arn:aws:sns:${var.sns["region"]}:${var.sns["account-id"]}:${var.sns["name"]}",
    ]

    sid = "__default_statement_ID"
  }

  statement {
    actions = [
      "SNS:Subscribe",
      "SNS:Receive",
    ]

    condition {
      test     = "StringLike"
      variable = "SNS:Endpoint"

      values = [
        "arn:aws:sqs:${var.sqs["region"]}:${var.sqs["account-id"]}:${var.sqs["name"]}",
      ]
    }

    effect = "Allow"

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    resources = [
      "arn:aws:sns:${var.sns["region"]}:${var.sns["account-id"]}:${var.sns["name"]}",
    ]

    sid = "__console_sub_0"
  }
}

data "aws_iam_policy_document" "sqs-queue-policy" {
  policy_id = "arn:aws:sqs:${var.sqs["region"]}:${var.sqs["account-id"]}:${var.sqs["name"]}/SQSDefaultPolicy"

  statement {
    sid    = "example-sns-topic"
    effect = "Allow"

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    actions = [
      "SQS:SendMessage",
    ]

    resources = [
      "arn:aws:sqs:${var.sqs["region"]}:${var.sqs["account-id"]}:${var.sqs["name"]}",
    ]

    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"

      values = [
        "arn:aws:sns:${var.sns["region"]}:${var.sns["account-id"]}:${var.sns["name"]}",
      ]
    }
  }
}

# provider to manage SNS topics
provider "aws" {
  alias  = "sns"
  region = "${var.sns["region"]}"

  assume_role {
    role_arn     = "arn:aws:iam::${var.sns["account-id"]}:role/${var.sns["role-name"]}"
    session_name = "sns-${var.sns["region"]}"
  }
}

# provider to manage SQS queues
provider "aws" {
  alias  = "sqs"
  region = "${var.sqs["region"]}"

  assume_role {
    role_arn     = "arn:aws:iam::${var.sqs["account-id"]}:role/${var.sqs["role-name"]}"
    session_name = "sqs-${var.sqs["region"]}"
  }
}

# provider to subscribe SQS to SNS (using the SQS account but the SNS region)
provider "aws" {
  alias  = "sns2sqs"
  region = "${var.sns["region"]}"

  assume_role {
    role_arn     = "arn:aws:iam::${var.sqs["account-id"]}:role/${var.sqs["role-name"]}"
    session_name = "sns2sqs-${var.sns["region"]}"
  }
}

resource "aws_sns_topic" "sns-topic" {
  provider     = "aws.sns"
  name         = "${var.sns["name"]}"
  display_name = "${var.sns["display_name"]}"
  policy       = "${data.aws_iam_policy_document.sns-topic-policy.json}"
}

resource "aws_sqs_queue" "sqs-queue" {
  provider = "aws.sqs"
  name     = "${var.sqs["name"]}"
  policy   = "${data.aws_iam_policy_document.sqs-queue-policy.json}"
}

resource "aws_sns_topic_subscription" "sns-topic" {
  provider  = "aws.sns2sqs"
  topic_arn = "${aws_sns_topic.sns-topic.arn}"
  protocol  = "sqs"
  endpoint  = "${aws_sqs_queue.sqs-queue.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `topic_arn` - (Required) The ARN of the SNS topic to subscribe to
* `protocol` - (Required) The protocol to use. The possible values for this are: `sqs`,  `lambda`, `application`. (`http` or `https` are partially supported, see below) (`email`, `sms`, are options but unsupported, see below).
* `endpoint` - (Required) The endpoint to send data to, the contents will vary with the protocol. (see below for more information)
* `endpoint_auto_confirms` - (Optional) Boolean indicating whether the end point is capable of [auto confirming subscription](http://docs.aws.amazon.com/sns/latest/dg/SendMessageToHttp.html#SendMessageToHttp.prepare) e.g., PagerDuty (default is false)
* `confirmation_timeout_in_minutes` - (Optional) Integer indicating number of minutes to wait in retying mode for fetching subscription arn before marking it as failure. Only applicable for http and https protocols (default is 1 minute).
* `raw_message_delivery` - (Optional) Boolean indicating whether or not to enable raw message delivery (the original message is directly passed, not wrapped in JSON with the original message in the message property).

### Protocols supported

Supported SNS protocols include:

* `lambda` -- delivery of JSON-encoded message to a lambda function
* `sqs` -- delivery of JSON-encoded message to an Amazon SQS queue
* `application` -- delivery of JSON-encoded message to an EndpointArn for a mobile app and device

Partially supported SNS protocols include:

* `http` -- delivery of JSON-encoded messages via HTTP. Supported only for the end points that auto confirms the subscription.
* `https` -- delivery of JSON-encoded messages via HTTPS. Supported only for the end points that auto confirms the subscription.

Unsupported protocols include the following:

* `email` -- delivery of message via SMTP
* `email-json` -- delivery of JSON-encoded message via SMTP
* `sms` -- delivery text message

These are unsupported because the endpoint needs to be authorized and does not
generate an ARN until the target email address has been validated. This breaks
the Terraform model and as a result are not currently supported.

### Specifying endpoints

Endpoints have different format requirements according to the protocol that is chosen.

* SQS endpoints come in the form of the SQS queue's ARN (not the URL of the queue) e.g: `arn:aws:sqs:us-west-2:432981146916:terraform-queue-too`
* Application endpoints are also the endpoint ARN for the mobile app and device.

## Attributes Reference

The following attributes are exported:

* `id` - The ARN of the subscription
* `topic_arn` - The ARN of the topic the subscription belongs to
* `protocol` - The protocol being used
* `endpoint` - The full endpoint to send data to (SQS ARN, HTTP(S) URL, Application ARN, SMS number, etc.)
* `arn` - The ARN of the subscription stored as a more user-friendly property

## Import

SNS Topic Subscriptions can be imported using the `subscription arn`, e.g.

```
$ terraform import aws_sns_topic_subscription.user_updates_sqs_target arn:aws:sns:us-west-2:0123456789012:my-topic:8a21d249-4329-4871-acc6-7be709c6ea7f
```
