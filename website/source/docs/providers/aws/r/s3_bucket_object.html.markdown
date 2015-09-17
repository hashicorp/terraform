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
* `source` - (Required) The path to the source file being uploaded to the bucket.
* `cache_control` - (Optional) Specifies caching behavior along the request/reply chain.
* `content_disposition` - (Optional) Specifies presentational information for the object.
* `content_encoding` - (Optional) Specifies what content encodings have been applied to the object and thus what decoding mechanisms must be applied to obtain the media-type referenced by the Content-Type header field.
* `content_language` - (Optional) The language the content is in.
* `content_type` - (Optional) A standard MIME type describing the format of the object data.

## Attributes Reference

The following attributes are exported

* `id` - the `key` of the resource supplied above
* `etag` - the ETag generated for the object. This is often the MD5 hash of the
object, unless you specify your own encryption keys
