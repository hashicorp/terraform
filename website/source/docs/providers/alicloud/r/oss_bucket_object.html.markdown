---
layout: "alicloud"
page_title: "Alicloud: alicloud_oss_bucket_object"
sidebar_current: "docs-alicloud-resource-oss-bucket-object"
description: |-
  Provides a resource to create a oss bucket object.
---

# alicloud\_oss\_bucket\_object

Provides a resource to put a object(content or file) to a oss bucket.

## Example Usage

### Uploading a file to a bucket

```
resource "alicloud_oss_bucket_object" "object-source" {
  bucket = "your_bucket_name"
  key    = "new_object_key"
  source = "path/to/file"
}
```

### Uploading a content to a bucket

```
resource "alicloud_oss_bucket" "example" {
  bucket = "your_bucket_name"
  acl = "public-read"
}

resource "alicloud_oss_bucket_object" "object-content" {
  bucket = "${alicloud_oss_bucket.example.bucket}"
  key    = "new_object_key"
  content = "the content that you want to upload."
}
```

## Argument Reference

-> **Note:** If you specify `content_encoding` you are responsible for encoding the body appropriately (i.e. `source` and `content` both expect already encoded/compressed bytes)

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to put the file in.
* `key` - (Required) The name of the object once it is in the bucket.
* `source` - (Required) The path to the source file being uploaded to the bucket.
* `content` - (Required unless `source` given) The literal content being uploaded to the bucket.
* `acl` - (Optional) The [canned ACL](https://help.aliyun.com/document_detail/31843.html?spm=5176.doc31842.2.2.j7C2nn) to apply. Defaults to "private".
* `content_type` - (Optional) A standard MIME type describing the format of the object data, e.g. application/octet-stream. All Valid MIME Types are valid for this input.
* `cache_control` - (Optional) Specifies caching behavior along the request/reply chain. Read [RFC2616 Cache-Control](https://www.ietf.org/rfc/rfc2616.txt?spm=5176.doc31978.2.1.iLEoOM&file=rfc2616.txt) for further details.
* `content_disposition` - (Optional) Specifies presentational information for the object. Read [RFC2616 Content-Disposition](https://www.ietf.org/rfc/rfc2616.txt?spm=5176.doc31978.2.1.iLEoOM&file=rfc2616.txt) for further details.
* `content_encoding` - (Optional) Specifies what content encodings have been applied to the object and thus what decoding mechanisms must be applied to obtain the media-type referenced by the Content-Type header field. Read [RFC2616 Content-Encoding](https://www.ietf.org/rfc/rfc2616.txt?spm=5176.doc31978.2.1.iLEoOM&file=rfc2616.txt) for further details.
* `content_md5` - (Optional) The MD5 value of the content. Read [MD5](https://help.aliyun.com/document_detail/31978.html?spm=5176.product31815.6.861.upTmI0) for computing method.
* `expires` - (Optional) Specifies expire date for the the request/response. Read [RFC2616 Expires](https://www.ietf.org/rfc/rfc2616.txt?spm=5176.doc31978.2.1.iLEoOM&file=rfc2616.txt) for further details.
* `server_side_encryption` - (Optional) Specifies server-side encryption of the object in OSS. At present, it valid value is "`AES256`".

Either `source` or `content` must be provided to specify the bucket content.
These two arguments are mutually-exclusive.

## Attributes Reference

The following attributes are exported

* `id` - the `key` of the resource supplied above
* `content_length` - the content length of request.
* `etag` - the ETag generated for the object (an MD5 sum of the object content).
