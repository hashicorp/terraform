---
layout: "google"
page_title: "Google: google_storage_bucket_object"
sidebar_current: "docs-google-storage-bucket-object"
description: |-
  Creates a new object inside a specified bucket
---

# google\_storage\_bucket\_object

Creates a new object inside an exisiting bucket in Google cloud storage service (GCS). Currently, it does not support creating custom ACLs. For more information see [the official documentation](https://cloud.google.com/storage/docs/overview) and [API](https://cloud.google.com/storage/docs/json_api).


## Example Usage

Example creating a public object in an existing `image-store` bucket.

```js
resource "google_storage_bucket_object" "picture" {
  name   = "butterfly01"
  source = "/images/nature/garden-tiger-moth.jpg"
  bucket = "image-store"
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the containing bucket.

* `name` - (Required) The name of the object.

- - -

* `content` - (Optional) Data as `string` to be uploaded. Must be defined if
    `source` is not.

* `predefined_acl` - (Optional, Deprecated) The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control#predefined-acl) apply. Please switch
to `google_storage_object_acl.predefined_acl`.


* `source` - (Optional) A path to the data you want to upload. Must be defined
    if `content` is not.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `crc32c` - (Computed) Base 64 CRC32 hash of the uploaded data.

* `md5hash` - (Computed) Base 64 MD5 hash of the uploaded data.
