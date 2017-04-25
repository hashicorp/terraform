---
layout: "aws"
page_title: "AWS: aws_config_configuration_recorder_status"
sidebar_current: "docs-aws-resource-config-configuration-recorder-status"
description: |-
  Manages status of an AWS Config Configuration Recorder.
---

# aws\_config\_configuration\_recorder\_status

Manages status (recording / stopped) of an AWS Config Configuration Recorder.

~> **Note:** Starting Configuration Recorder requires a [Delivery Channel](/docs/providers/aws/r/config_delivery_channel.html) to be present. Use of `depends_on` (as shown below) is recommended to avoid race conditions.

## Example Usage

```hcl
resource "aws_config_configuration_recorder_status" "foo" {
  name       = "${aws_config_configuration_recorder.foo.name}"
  is_enabled = true
  depends_on = ["aws_config_delivery_channel.foo"]
}

resource "aws_iam_role_policy_attachment" "a" {
  role       = "${aws_iam_role.r.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSConfigRole"
}

resource "aws_s3_bucket" "b" {
  bucket = "awsconfig-example"
}

resource "aws_config_delivery_channel" "foo" {
  name           = "example"
  s3_bucket_name = "${aws_s3_bucket.b.bucket}"
}

resource "aws_config_configuration_recorder" "foo" {
  name     = "example"
  role_arn = "${aws_iam_role.r.arn}"
}

resource "aws_iam_role" "r" {
  name = "example-awsconfig"

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

* `name` - (Required) The name of the recorder
* `is_enabled` - (Required) Whether the configuration recorder should be enabled or disabled.

## Import

Configuration Recorder Status can be imported using the name of the Configuration Recorder, e.g.

```
$ terraform import aws_config_configuration_recorder_status.foo example
```
