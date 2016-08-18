---
layout: "aws"
page_title: "AWS: sns_application"
sidebar_current: "docs-aws-resource-sns-application"
description: |-
  Provides an SNS application resource.
---

# aws\_sns\_application

Provides an SNS application resource

## Example Usage

```
resource "aws_sns_application" "gcm_application" {
  resource "aws_sns_application" "gcm_application" {
  	name = "aws_sns_gcm_application"
  	platform = "GCM"
  	credential = "<GCM API KEY>"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The friendly name for the SNS application
* `platform` - (Required) The platform that the app is registered with. See [Platform][1] for supported platforms.
* `credential` - (Required) Application Platform credential. See [Credential][1] for type of credential required for platform.
* `principal` - (Optional) Application Platform principal. See [Principal][2] for type of principal required for platform.
* `created_topic` - (Optional) SNS Topic triggered when a new platform endpoint is added to your platform application.
* `deleted_topic` - (Optional) SNS Topic triggered when any of the platform endpoints associated with your platform application is deleted.
* `updated_topic` - (Optional) SNS Topic triggered when any of the attributes of the platform endpoints associated with your platform application are changed.
* `failure_topic` - (Optional) SNS Topic triggered when a delivery to any of the platform endpoints associated with your platform application encounters a permanent failure.
* `success_iam_role` - (Optional) The IAM role permitted to receive success feedback for this application.
* `failure_iam_role` - (Optional) The percentage of success to sample (0-100)
* `success_sample_rate` - (Optional) The IAM role permitted to receive failure feedback for this application.

## Platforms supported

Supported SNS Application Platforms include

* `APNS` - Apple iOS Push Notification Service
* `APNS_SANDBOX` - Apple iOS Push Notification Service Development
* `APNS_VOIP` - Apple VOIP Push Notification Service
* `APNS_VOIP_SANDBOX` - Apple VOIP Push Notification Service Development
* `MACOS` - Apple MacOS Push Notification Service
* `MACOS_SANDBOX` - Apple MacOS Push Notification Service Development
* `GCM` - Google Cloud Messaging

Unsupported SNS Application Platforms include

* `ADM` - Amazon Device Messaging
* `Baidu` - Baidu Cloud Push for Android in China
* `MPNS` - Microsoft MPNS for Windows Phone 7+
* `WNS` - Microsoft WNS for Windows 8+ & Windows Phone 8.1+

## Attributes Reference

The following attributes are exported:

* `id` - The ARN of the SNS application
* `arn` - The ARN of the SNS application, as a more obvious property (clone of id)

[1]: http://docs.aws.amazon.com/sns/latest/dg/mobile-push-send-register.html
[2]: http://docs.aws.amazon.com/sns/latest/api/API_CreatePlatformApplication.html

## Import

SNS Applications can be imported using the `application arn`, e.g.

```
$ terraform import aws_sns_application.gcm_application arn:aws:sns:us-west-2:0123456789012:app/GCM/aws_sns_gcm_application
```
