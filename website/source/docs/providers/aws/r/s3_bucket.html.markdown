---
layout: "aws"
page_title: "AWS: aws_s3_bucket"
sidebar_current: "docs-aws-resource-s3-bucket"
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
* `acl` - (Optional) The canned ACL to apply. Defaults to "private".

## Attributes Reference

The following attributes are exported:

* `id` - The name of the bucket

