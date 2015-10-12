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
* `content` - (Required unless `source` given) The literal content being uploaded to the bucket.
* `cache_control` - (Optional) Specifies caching behavior along the request/reply chain Read [w3c cache_control](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.9) for futher details.
* `content_disposition` - (Optional) Specifies presentational information for the object. Read [wc3 content_disposition](http://www.w3.org/Protocols/rfc2616/rfc2616-sec19.html#sec19.5.1) for further information.
* `content_encoding` - (Optional) Specifies what content encodings have been applied to the object and thus what decoding mechanisms must be applied to obtain the media-type referenced by the Content-Type header field. Read [w3c content encoding](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.11) for further information.
* `content_language` - (Optional) The language the content is in e.g. en-US or en-GB.
* `content_type` - (Optional) A standard MIME type describing the format of the object data, e.g. application/octet-stream. All Valid MIME Types are valid for this input.

Either `source` or `content` must be provided to specify the bucket content.
These two arguments are mutually-exclusive.

## Attributes Reference

The following attributes are exported

* `id` - the `key` of the resource supplied above
* `etag` - the ETag generated for the object. This is often the MD5 hash of the
object, unless you specify your own encryption keys
