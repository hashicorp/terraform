---
layout: "aws"
page_title: "AWS: aws_config_configuration_recorder"
sidebar_current: "docs-aws-resource-config-configuration-recorder"
description: |-
  Provides an AWS Config Configuration Recorder.
---

# aws\_config\_configuration\_recorder

Provides an AWS Config Configuration Recorder. Please note that this resource **does not start** the created recorder automatically.

~> **Note:** _Starting_ the Configuration Recorder requires a [delivery channel](/docs/providers/aws/r/config_delivery_channel.html) (while delivery channel creation requires Configuration Recorder). This is why [`aws_config_configuration_recorder_status`](/docs/providers/aws/r/config_configuration_recorder_status.html) is a separate resource.

## Example Usage

```hcl
resource "aws_config_configuration_recorder" "foo" {
  name     = "example"
  role_arn = "${aws_iam_role.r.arn}"
}

resource "aws_iam_role" "r" {
  name = "awsconfig-example"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the recorder. Defaults to `default`
* `role_arn` - (Required) Amazon Resource Name (ARN) of the IAM role
	used to make read or write requests to the delivery channel and to describe the AWS resources associated with the account.
	See [AWS Docs](http://docs.aws.amazon.com/config/latest/developerguide/iamrole-permissions.html) for more details.
* `recording_group` - (Optional) Recording group - see below.

### `recording_group`

* `all_supported` - (Optional) Specifies whether AWS Config records configuration changes
	for every supported type of regional resource (which includes any new type that will become supported in the future).
	Conflicts with `resource_types`. Defaults to `true`.
* `include_global_resource_types` - (Optional) Specifies whether AWS Config includes all supported types of *global resources*
	with the resources that it records. Requires `all_supported = true`. Conflicts with `resource_types`.
* `resource_types` - (Optional) A list that specifies the types of AWS resources for which
	AWS Config records configuration changes (for example, `AWS::EC2::Instance` or `AWS::CloudTrail::Trail`).
  See [relevant part of AWS Docs](http://docs.aws.amazon.com/config/latest/APIReference/API_ResourceIdentifier.html#config-Type-ResourceIdentifier-resourceType) for available types.

## Attributes Reference

The following attributes are exported:

* `id` - Name of the recorder

## Import

Configuration Recorder can be imported using the name, e.g.

```
$ terraform import aws_config_configuration_recorder.foo example
```
