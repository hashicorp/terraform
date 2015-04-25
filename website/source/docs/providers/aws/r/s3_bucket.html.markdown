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

    tags {
        Name = "My bucket"
        Environment = "Dev"
    }
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket.
* `acl` - (Optional) The [canned ACL](http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl) to apply. Defaults to "private".
* `tags` - (Optional) A mapping of tags to assign to the bucket.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the bucket

