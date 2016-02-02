+---
 +layout: "aws"
 +page_title: "AWS: aws_sns_platform_application_apns"
 +sidebar_current: "docs-aws-sns-platform-application-apns"
 +description: |-
 +  Provides an Apple Push Notification Services Platform Application
 +---
 +

# aws\_sns\_platform\_application\_apns

Provides an SNS platform application resource for Apple Push Notfication Services

## Example Usage

```

resource "aws_sns_platform_application_apns" "platform_application_ios" {
  type = "SANDBOX"
  name = "platform_application_ios"
  platform_credential = "${file("certs/development_com.yourapplication.pkey")}"
  platform_principal = "${file("certs/development_com.yourapplication.pem")}"
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The friendly name for the platform endpoint
* `type` - (Optional) 'SANDBOX' for development
* `platform_credential` - (Required) Your APNS private key
* `platform_principal` - (Required) Your APNS cert

## Attributes Reference

The following attributes are exported:

* `id` - The ARN of the platform application

