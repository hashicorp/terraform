---
layout: "aws"
page_title: "AWS: aws_devicefarm_project"
sidebar_current: "docs-aws-resource-devicefarm-project"
description: |-
  Provides a DeviceFarm Project.
---

# aws\_devicefarm\_project

Provides a DeviceFarm Project.

## Example Usage

```
resource "aws_devicefarm_project" "example" {
    name = "tf-testproject-01
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DeviceFarm project.

## Attributes Reference

The following attributes are exported:

* `arn` - The ARN of the DeviceFarm Project