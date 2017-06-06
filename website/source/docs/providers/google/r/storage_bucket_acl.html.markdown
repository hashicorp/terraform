---
layout: "google"
page_title: "Google: google_storage_bucket_acl"
sidebar_current: "docs-google-storage-bucket-acl"
description: |-
  Creates a new bucket ACL in Google Cloud Storage.
---

# google\_storage\_bucket\_acl

Creates a new bucket ACL in Google cloud storage service (GCS). For more information see 
[the official documentation](https://cloud.google.com/storage/docs/access-control/lists) 
and 
[API](https://cloud.google.com/storage/docs/json_api/v1/bucketAccessControls).

## Example Usage

Example creating an ACL on a bucket with one owner, and one reader.

```hcl
resource "google_storage_bucket" "image-store" {
  name     = "image-store-bucket"
  location = "EU"
}

resource "google_storage_bucket_acl" "image-store-acl" {
  bucket = "${google_storage_bucket.image-store.name}"

  role_entity = [
    "OWNER:user-my.email@gmail.com",
    "READER:group-mygroup",
  ]
}
```

## Argument Reference

* `bucket` - (Required) The name of the bucket it applies to.

- - -

* `predefined_acl` - (Optional) The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control/lists#predefined-acl) to apply. Must be set if `role_entity` is not.

* `role_entity` - (Optional) List of role/entity pairs in the form `ROLE:entity`. See [GCS Bucket ACL documentation](https://cloud.google.com/storage/docs/json_api/v1/bucketAccessControls)  for more details. Must be set if `predefined_acl` is not.

## Attributes Reference

Only the arguments listed above are exposed as attributes.
