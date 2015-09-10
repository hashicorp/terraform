---
layout: "aws"
page_title: "AWS: aws_s3_bucket_object"
side_bar_current: "docs-aws-resource-s3-bucket-object"
description: |-
  Provides a S3 bucket object resource.
---

# aws\_s3\_bucket\_object

Provides a S3 bucket object resource.

## Example Usage

### Uploading a file to a bucket

```
resource "aws_s3_bucket_object" "object" {
	bucket = "your_bucket_name"
	key = "new_object_key"
	source = "path/to/file"
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to put the file in.
* `key` - (Required) The name of the object once it is in the bucket.
* `source` - (Required unless `content` given) The path to the source file being uploaded to the bucket.
* `content` - (Required unless `source` given) The literal content being uploaded to the bucket.

Either `source` or `content` must be provided to specify the bucket content.
These two arguments are mutually-exclusive.

## Attributes Reference

The following attributes are exported

* `id` - the `key` of the resource supplied above
* `etag` - the ETag generated for the object. This is often the MD5 hash of the
object, unless you specify your own encryption keys
