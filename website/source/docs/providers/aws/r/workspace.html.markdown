---
layout: "aws"
page_title: "AWS: aws_workspace"
sidebar_current: "docs-aws-resource-workspace"
description: |-
  Provides a workspace in AWS Workspaces Service.
---

# aws\_workspace

Provides a workspace in AWS Workspaces Service

## Example Usage

```
resource "aws_directory_service_directory" "corp_directory" {
  name     = "corp.notexample.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
}

resource "aws_workspace" "foo" {
  bundle_id    = "bar"
  directory_id = ${aws_directory_service_directory.corp_directory.id}
  user_name    = "Administrator"
}
```

## Argument Reference

The following arguments are supported:
* `bundle_name` - (Optional) The name of bundle that the WorkSpace is created from. Required if not specifying `bundle_id`
* `bundle_id` - (Optional) The identifier of a bundle that the WorkSpace is created from. Required if not specifying `bundle_name`
* `directory_id` - (Required) The identifier of the AWS Directory Service directory that the WorkSpace belongs to.
* `root_volume_encryption` - (Required) Specifies whether the data stored on the root volume, or C: drive, is encrypted.
* `user_name` - (Required) The user that the WorkSpace is assigned to.
* `user_volume_encryption` - (Required) Specifies whether the data stored on the user volume, or D: drive, is encrypted.
* `volume_encryption_key` - (Required) The KMS key used to encrypt data stored on your WorkSpace.

## Attributes Reference

The following attributes are exported:
* `id` - The WorkSpace identifier.
* `computer_name` - The name of the WorksSpace as seen by the operating system.
* `ip_address` - The IP address of the WorkSpace.
* `subnet_id` - The indentifier of the subnet that the WorkSpace is in.
