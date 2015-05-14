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

### Private Bucket w/ Tags

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

### Static Website Hosting

```
resource "aws_s3_bucket" "b" {
    bucket = "s3-website-test.hashicorp.com"
    acl = "public-read"

    website {
        index_document = "index.html"
        error_document = "error.html"
    }
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket.
* `acl` - (Optional) The [canned ACL](http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl) to apply. Defaults to "private".
* `tags` - (Optional) A mapping of tags to assign to the bucket.
* `website` - (Optional) A website object (documented below).

The website object supports the following:

* `index_document` - (Required) Amazon S3 returns this index document when requests are made to the root domain or any of the subfolders.
* `error_document` - (Optional) An absolute path to the document to return in case of a 4XX error.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the bucket.
* `hosted_zone_id` - The [Route 53 Hosted Zone ID](http://docs.aws.amazon.com/general/latest/gr/rande.html#s3_website_region_endpoints) for this bucket's region.
* `region` - The AWS region this bucket resides in.
* `website_endpoint` - The website endpoint, if the bucket is configured with a website. If not, this will be an empty string.
