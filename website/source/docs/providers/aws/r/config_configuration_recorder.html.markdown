---
layout: "aws"
page_title: "AWS: aws_config_configuration_recorder"
sidebar_current: "docs-aws-resource-config-configuration-recorder"
description: |-
  Provides an AWS Config Configuration Recorder.
---

# aws\_config\_configuration\_recorder

Provides an AWS Config Configuration Recorder.

~> It is impossible to start the recorder without having a delivery channel. It is therefore recommended to add `depends_on = ["aws_config_delivery_channel.xzy"]` to prevent race conditions.

## Example Usage

```
resource "aws_config_configuration_recorder" "foo" {
  name = "michael-s-rogers"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the recorder. Defaults to `default`
* `is_enabled` - (Optional) Whether to enable the recorderding. Defaults to `true`
* `role_arn` - (Optional) Amazon Resource Name (ARN) of the IAM role
	used to make read or write requests to the delivery channel
	and to describe the AWS resources associated with the account.
	See [AWS Docs](http://docs.aws.amazon.com/config/latest/developerguide/iamrole-permissions.html) for more details.
* `recording_group` - (Optional) Recording group - see below.

`recording_group` supports the following:

* `all_supported` - (Optional) Specifies whether AWS Config records configuration changes
	for every supported type of regional resource (which includes any new type that will become supported in the future).
	Conflicts with `resource_types`.
* `include_global_resource_types` - (Optional) Specifies whether AWS Config includes all supported types of *global resources*
	with the resources that it records. Requires `all_supported = true`. Conflicts with `resource_types`.
* `resource_types` - (Optional) A list that specifies the types of AWS resources for which
	AWS Config records configuration changes (for example, `AWS::EC2::Instance` or `AWS::CloudTrail::Trail`).

## Attributes Reference

The following attributes are exported:

* `id` - Name of the recorder
