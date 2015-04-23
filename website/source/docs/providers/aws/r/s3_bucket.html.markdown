---
layout: "aws"
page_title: "AWS: aws_s3_bucket"
sidebar_current: "docs-aws-resource-s3-bucket"
description: |-
  Provides a S3 bucket resource.
---

# aws\_s3\_bucket

Provides a S3 bucket resource.

## Example Usage

```
resource "aws_s3_bucket" "b" {
    bucket = "my_tf_test_bucket"
    acl = "private"
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket.
* `acl` - (Optional) The canned ACL to apply.
* `grant_full_control` - (Optional) The grantee which has full controll permission.
* `grant_read` - (Optional) The grantee which has read permission.
* `grant_read_acp` - (Optional) The grantee which has read ACP permission.
* `grant_write` - (Optional) The grantee which has write permission.
* `grant_write_acp` - (Optional) The grantee which has write ACP permission.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the bucket

