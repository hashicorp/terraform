---
layout: "aws"
page_title: "AWS: aws_s3_bucket_policy"
sidebar_current: "docs-aws-resource-s3-bucket-policy"
description: |-
  Attaches a policy to an S3 bucket resource.
---

# aws\_s3\_bucket\_policy

Attaches a policy to an S3 bucket resource.

## Example Usage

### Using versioning

```hcl
resource "aws_s3_bucket" "b" {
  # Arguments
}

data "aws_iam_policy_document" "b" {
  # Policy statements
}

resource "aws_s3_bucket_policy" "b" {
  bucket = "${aws_s3_bucket.b.id}"
  policy = "${data.aws_iam_policy_document.b.json}"
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to which to apply the policy.
* `policy` - (Required) The text of the policy.
