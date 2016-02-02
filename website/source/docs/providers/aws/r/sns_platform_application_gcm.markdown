+---
 +layout: "aws"
 +page_title: "AWS: aws_sns_platform_application_gcm"
 +sidebar_current: "docs-aws-sns-platform-application-gcm"
 +description: |-
 +  Provides a Google Cloud Messaging Platform Application
 +---
 +

# aws\_sns\_platform\_application\_gcm

Provides an SNS platform application resource for Google Cloud Messaging

## Example Usage

```
resource "aws_sns_platform_application_gcm" "platform_application_android" {
  name = "platform_application_android"
  platform_credential = "AIzaSaBTABlY4JdtF03bYgH9knXdZJUjgkIQ5ks"
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The friendly name for the platform endpoint
* `platform_credential` - (Required) The Google application API Key

## Attributes Reference

The following attributes are exported:

* `id` - The ARN of the platform application

