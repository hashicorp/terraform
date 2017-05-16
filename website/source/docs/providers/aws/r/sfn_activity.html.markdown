---
layout: "aws"
page_title: "AWS: sfn_activity"
sidebar_current: "docs-aws-resource-sfn-activity"
description: |-
  Provides a Step Function Activity resource.
---

# sfn\_activity

Provides a Step Function Activity resource

## Example Usage

```hcl
resource "aws_sfn_activity" "sfn_activity" {
  name = "my-activity"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the activity to create.

## Attributes Reference

The following attributes are exported:

* `id` - The Amazon Resource Name (ARN) that identifies the created activity.
* `name` - The name of the activity.
* `creation_date` - The date the activity was created.

## Import

Activities can be imported using the `arn`, e.g.

```
$ terraform import aws_sfn_activity.foo arn:aws:states:eu-west-1:123456789098:activity:bar
```
