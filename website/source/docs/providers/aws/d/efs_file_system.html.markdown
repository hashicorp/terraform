---
layout: "aws"
page_title: "AWS: efs_file_system"
sidebar_current: "docs-aws-datasource-efs-file-system"
description: |-
  Provides an Elastic File System (EFS) data source.
---

# aws_efs_file_system

Provides information about an Elastic File System (EFS).

## Example Usage

```hcl
variable "file_system_id" {
  type = "string"
  default = ""
}

data "aws_efs_file_system" "by_id" {
  file_system_id = "${var.file_system_id}"
}
```

## Argument Reference

The following arguments are supported:

* `file_system_id` - (Optional) The ID that identifies the file system (e.g. fs-ccfc0d65).
* `creation_token` - (Optional) Restricts the list to the file system with this creation token

## Attributes Reference

The following attributes are exported:

* `performance_mode` - The PerformanceMode of the file system.
* `tags` - The list of tags assigned to the file system.

