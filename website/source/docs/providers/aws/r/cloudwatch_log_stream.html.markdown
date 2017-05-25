---
layout: "aws"
page_title: "AWS: aws_cloudwatch_log_stream"
sidebar_current: "docs-aws-resource-cloudwatch-log-stream"
description: |-
  Provides a CloudWatch Log Stream resource.
---

# aws\_cloudwatch\_log\_stream

Provides a CloudWatch Log Stream resource.

## Example Usage

```hcl
resource "aws_cloudwatch_log_group" "yada" {
  name = "Yada"
}

resource "aws_cloudwatch_log_stream" "foo" {
  name           = "SampleLogStream1234"
  log_group_name = "${aws_cloudwatch_log_group.yada.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the log stream. Must not be longer than 512 characters and must not contain `:`
* `log_group_name` - (Required) The name of the log group under which the log stream is to be created.

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) specifying the log stream.