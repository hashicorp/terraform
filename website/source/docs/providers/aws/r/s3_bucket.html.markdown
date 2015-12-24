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
    policy = "${file("policy.json")}"

    website {
        index_document = "index.html"
        error_document = "error.html"
    }
}
```

### Using CORS

```
resource "aws_s3_bucket" "b" {
    bucket = "s3-website-test.hashicorp.com"
    acl = "public-read"

    cors_rule {
        allowed_headers = ["*"]
        allowed_methods = ["PUT","POST"]
        allowed_origins = ["https://s3-website-test.hashicorp.com"]
        expose_headers = ["ETag"]
        max_age_seconds = 3000
    }
}
```

### Using versioning

```
resource "aws_s3_bucket" "b" {
    bucket = "my_tf_test_bucket"
    acl = "private"
    versioning {
        enabled = true
    }
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket.
* `acl` - (Optional) The [canned ACL](http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl) to apply. Defaults to "private".
* `policy` - (Optional) A valid [bucket policy](http://docs.aws.amazon.com/AmazonS3/latest/dev/example-bucket-policies.html) JSON document. Note that if the policy document is not specific enough (but still valid), Terraform may view the policy as constantly changing in a `terraform plan`. In this case, please make sure you use the verbose/specific version of the policy.

* `tags` - (Optional) A mapping of tags to assign to the bucket.
* `force_destroy` - (Optional, Default:false ) A boolean that indicates all objects should be deleted from the bucket so that the bucket can be destroyed without error. These objects are *not* recoverable.
* `website` - (Optional) A website object (documented below).
* `cors_rule` - (Optional) A rule of [Cross-Origin Resource Sharing](http://docs.aws.amazon.com/AmazonS3/latest/dev/cors.html) (documented below).
* `versioning` - (Optional) A state of [versioning](http://docs.aws.amazon.com/AmazonS3/latest/dev/Versioning.html) (documented below)

The website object supports the following:

* `index_document` - (Required, unless using `redirect_all_requests_to`) Amazon S3 returns this index document when requests are made to the root domain or any of the subfolders.
* `error_document` - (Optional) An absolute path to the document to return in case of a 4XX error.
* `redirect_all_requests_to` - (Optional) A hostname to redirect all website requests for this bucket to.

The CORS supports the following:

* `allowed_headers` (Optional) Specifies which headers are allowed.
* `allowed_methods` (Required) Specifies which methods are allowed. Can be `GET`, `PUT`, `POST`, `DELETE` or `HEAD`.
* `allowed_origins` (Required) Specifies which origins are allowed.
* `expose_headers` (Optional) Specifies expose header in the response.
* `max_age_seconds` (Optional) Specifies time in seconds that browser can cache the response for a preflight request.

The versioning supports the following:

* `enabled` - (Optional) Enable versioning. Once you version-enable a bucket, it can never return to an unversioned state. You can, however, suspend versioning on that bucket.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the bucket.
* `arn` - The ARN of the bucket. Will be of format `arn:aws:s3:::bucketname`
* `hosted_zone_id` - The [Route 53 Hosted Zone ID](http://docs.aws.amazon.com/general/latest/gr/rande.html#s3_website_region_endpoints) for this bucket's region.
* `region` - The AWS region this bucket resides in.
* `website_endpoint` - The website endpoint, if the bucket is configured with a website. If not, this will be an empty string.
* `website_domain` - The domain of the website endpoint, if the bucket is configured with a website. If not, this will be an empty string. This is used to create Route 53 alias records.
