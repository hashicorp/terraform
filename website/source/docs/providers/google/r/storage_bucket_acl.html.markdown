---
layout: "google"
page_title: "Google: google_storage_bucket_acl"
sidebar_current: "docs-google-storage-bucket-acl"
description: |-
  Creates a new bucket ACL in Google Cloud Storage.
---

# google\_storage\_bucket\_acl

Creates a new bucket ACL in Google cloud storage service(GCS). 

## Example Usage

Example creating an ACL on a bucket with one owner, and one reader.

```
resource "google_storage_bucket" "image-store" {
	name = "image-store-bucket"
	location = "EU"
}

resource "google_storage_bucket_acl" "image-store-acl" {
    bucket = "${google_storage_bucket.image_store.name}"
    role_entity = ["OWNER:user-my.email@gmail.com", 
        "READER:group-mygroup"]
}

```

## Argument Reference

* `bucket` - (Required) The name of the bucket it applies to.
* `predefined_acl` - (Optional) The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control#predefined-acl) to apply. Must be set if both `role_entity` and `default_acl` are not.
* `default_acl` - (Optional) The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control#predefined-acl) to apply to future buckets. Must be set both `role_entity` and `predefined_acl` are not.
* `role_entity` - (Optional) List of role/entity pairs in the form `ROLE:entity`. See [GCS Bucket ACL documentation](https://cloud.google.com/storage/docs/json_api/v1/bucketAccessControls)  for more details. Must be set if both `predefined_acl` and `default_acl` are not.
