---
layout: "aws"
page_title: "AWS: aws_s3_bucket_object"
sidebar_current: "docs-aws-datasource-s3-bucket-object"
description: |-
    Provides metadata and optionally content of an S3 object
---

# aws\_s3\_bucket\_object

The S3 object data source allows access to the metadata and
_optionally_ (see below) content of an object stored inside S3 bucket.

~> **Note:** The content of an object (`body` field) is available only for objects which have a human-readable `Content-Type` (`text/*` and `application/json`). This is to prevent printing unsafe characters and potentially downloading large amount of data which would be thrown away in favour of metadata.

## Example Usage

```hcl
data "aws_s3_bucket_object" "lambda" {
  bucket = "my-lambda-functions"
  key    = "hello-world.zip"
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "iam_for_lambda"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_lambda_function" "test_lambda" {
  s3_bucket         = "${data.aws_s3_bucket_object.lambda.bucket}"
  s3_key            = "${data.aws_s3_bucket_object.lambda.key}"
  s3_object_version = "${data.aws_s3_bucket_object.lambda.version_id}"
  function_name     = "lambda_function_name"
  role              = "${aws_iam_role.iam_for_lambda.arn}"
  handler           = "exports.test"
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to read the object from
* `key` - (Required) The full path to the object inside the bucket
* `version_id` - (Optional) Specific version ID of the object returned (defaults to latest version)

## Attributes Reference

The following attributes are exported:

* `body` - Object data (see **limitations above** to understand cases in which this field is actually available)
* `cache_control` - Specifies caching behavior along the request/reply chain.
* `content_disposition` - Specifies presentational information for the object.
* `content_encoding` - Specifies what content encodings have been applied to the object and thus what decoding mechanisms must be applied to obtain the media-type referenced by the Content-Type header field.
* `content_language` - The language the content is in.
* `content_length` - Size of the body in bytes.
* `content_type` - A standard MIME type describing the format of the object data.
* `etag` - [ETag](https://en.wikipedia.org/wiki/HTTP_ETag) generated for the object (an MD5 sum of the object content in case it's not encrypted)
* `expiration` - If the object expiration is configured (see [object lifecycle management](http://docs.aws.amazon.com/AmazonS3/latest/dev/object-lifecycle-mgmt.html)), the field includes this header. It includes the expiry-date and rule-id key value pairs providing object expiration information. The value of the rule-id is URL encoded.
* `expires` - The date and time at which the object is no longer cacheable.
* `last_modified` - Last modified date of the object in RFC1123 format (e.g. `Mon, 02 Jan 2006 15:04:05 MST`)
* `metadata` - A map of metadata stored with the object in S3
* `server_side_encryption` - If the object is stored using server-side encryption (KMS or Amazon S3-managed encryption key), this field includes the chosen encryption and algorithm used.
* `sse_kms_key_id` - If present, specifies the ID of the Key Management Service (KMS) master encryption key that was used for the object.
* `storage_class` - [Storage class](http://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html) information of the object. Available for all objects except for `Standard` storage class objects.
* `version_id` - The latest version ID of the object returned.
* `website_redirect_location` - If the bucket is configured as a website, redirects requests for this object to another object in the same bucket or to an external URL. Amazon S3 stores the value of this header in the object metadata.
* `tags`  - A mapping of tags assigned to the object.
