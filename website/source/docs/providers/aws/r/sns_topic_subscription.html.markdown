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

## Example Usage

You can directly supply a topic ARN by hand in the `topic_arn` property:

```
resource "aws_sns_topic_subscription" "user_updates_sqs_target" {
    topic_arn = "arn:aws:sns:us-west-2:432981146916:user-updates-topic"
    topic_arn = "${aws_sns_topic.user_updates.id}"
    protocol = "sqs"
    endpoint = "arn:aws:sqs:us-west-2:432981146916:terraform-queue-too"
}
```

Alternatively you can use the identifier of a previously created SNS topic:

```
resource "aws_sns_topic" "user_updates" {
  name = "user-updates-topic"
}

resource "aws_sns_topic_subscription" "user_updates_sqs_target" {
    topic_arn = "${aws_sns_topic.user_updates.id}"
    protocol = "sqs"
    endpoint = "arn:aws:sqs:us-west-2:432981146916:terraform-queue-too"
}
```


Currently there is no SQS support, so you need to know the queue ARN ahead of time, however it would make sense to be
able to populate the endpoint from an SQS resource in your JSON file.

## Argument Reference

The following arguments are supported:

* `topic_arn` - (Required) The ARN of the SNS topic to subscribe to
* `protocol` - (Required) The protocol to use. The possible values for this are: `sqs`, `http`, `https`, `sms`, or `application`. (`email` is an option but unsupported, see below)
* `endpoint` - (Required) The endpoint to send data to, the contents will vary with the protocol. (see below for more information)

### Protocols supported

Supported SNS protocols include:

* `http` -- delivery of JSON-encoded message via HTTP POST
* `https` -- delivery of JSON-encoded message via HTTPS POST
* `sms` -- delivery of message via SMS
* `sqs` -- delivery of JSON-encoded message to an Amazon SQS queue
* `application` -- delivery of JSON-encoded message to an EndpointArn for a mobile app and device

Unsupported protocols include the following:

* `email` -- delivery of message via SMTP
* `email-json` -- delivery of JSON-encoded message via SMTP

These are unsupported because the email address needs to be authorized and does not generate an ARN until the target email address has been validated. This breaks
the Terraform model and as a result are not currently supported.

### Specifying endpoints

Endpoints have different format requirements according to the protocol that is chosen.

* HTTP/HTTPS endpoints will require a URL to POST data to
* SMS endpoints are mobile numbers that are capable of receiving an SMS
* SQS endpoints come in the form of the SQS queue's ARN (not the URL of the queue) e.g: `arn:aws:sqs:us-west-2:432981146916:terraform-queue-too`
* Application endpoints are also the endpoint ARN for the mobile app and device.


## Attributes Reference

The following attributes are exported:

* `id` - The ARN of the subscription
* `topic_arn` - The ARN of the topic the subscription belongs to
* `protocol` - The protocol being used
* `endpoint` - The full endpoint to send data to (SQS ARN, HTTP(S) URL, Application ARN, SMS number, etc.)

