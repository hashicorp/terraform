---
layout: "aws"
page_title: "AWS: aws_ssm_parameter"
sidebar_current: "docs-aws-datasource-ssm-parameter"
description: |-
  Provides a SSM Parameter datasource
---

# aws\_ssm\_parameter

Provides an SSM Parameter data source.

## Example Usage

To store a basic string parameter:

```hcl
data "aws_ssm_parameter" "foo" {
  name  = "foo"
}
```

~> **Note:** The unencrypted value of a SecureString will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the parameter.


The following attributes are exported:

* `name` - (Required) The name of the parameter.
* `type` - (Required) The type of the parameter. Valid types are `String`, `StringList` and `SecureString`.
* `value` - (Required) The value of the parameter.
