---
layout: "aws"
page_title: "AWS: aws_cloudwatch_log_group"
sidebar_current: "docs-aws-resource-cloudwatch-log-group"
description: |-
  Provides a CloudWatch Log Group resource.
---

# aws\_cloudwatch\_log\_group

Provides a CloudWatch Log Group resource.

## Example Usage

```
resource "aws_cloudwatch_log_group" "yada" {
  name = "Yada"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the log group
* `retention_in_days` - (Optional) Specifies the number of days
  you want to retain log events in the specified log group.

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) specifying the log group.
