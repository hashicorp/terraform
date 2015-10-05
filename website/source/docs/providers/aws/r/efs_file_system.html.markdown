---
layout: "aws"
page_title: "AWS: aws_efs_file_system"
sidebar_current: "docs-aws-resource-efs-file-system"
description: |-
  Provides an EFS file system.
---

# aws\_efs\_file\_system

Provides an EFS file system.

## Example Usage

```
resource "aws_efs_file_system" "foo" {
  reference_name = "my-product"
  tags {
    Name = "MyProduct"
  }
}
```

## Argument Reference

The following arguments are supported:

* `reference_name` - (Optional) A reference name used in Creation Token
* `tags` - (Optional) A mapping of tags to assign to the file system

## Attributes Reference

The following attributes are exported:

* `id` - The ID that identifies the file system
