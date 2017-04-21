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

```hcl
resource "aws_cloudwatch_log_group" "yada" {
  name = "Yada"

  tags {
    Environment = "production"
    Application = "serviceA"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional, Forces new resource) The name of the log group. If omitted, Terraform will assign a random, unique name.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `retention_in_days` - (Optional) Specifies the number of days
  you want to retain log events in the specified log group.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name (ARN) specifying the log group.


## Import

Cloudwatch Log Groups can be imported using the `name`, e.g.

```
$ terraform import aws_cloudwatch_log_group.test_group yada
```
