---
layout: "google"
page_title: "Google: google_storage_object_acl"
sidebar_current: "docs-google-storage-object-acl"
description: |-
  Creates a new object ACL in Google Cloud Storage.
---

# google\_storage\_object\_acl

Creates a new object ACL in Google cloud storage service (GCS)

## Example Usage

Create an object ACL with one owner and one reader.

```js
resource "google_storage_bucket" "image-store" {
  name     = "image-store-bucket"
  location = "EU"
}

resource "google_storage_bucket_object" "image" {
  name  = "image1"
  bucket = "${google_storage_bucket.name}"
  source = "image1.jpg"
}

resource "google_storage_object_acl" "image-store-acl" {
  bucket = "${google_storage_bucket.image_store.name}"
  object = "${google_storage_bucket_object.image_store.name}"

  role_entity = [
    "OWNER:user-my.email@gmail.com",
    "READER:group-mygroup",
  ]
}
```

## Argument Reference

* `bucket` - (Required) The name of the bucket it applies to.

* `object` - (Required) The name of the object it applies to.

- - -

* `predefined_acl` - (Optional) The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control#predefined-acl) to apply. Must be set if `role_entity` is not.

* `role_entity` - (Optional) List of role/entity pairs in the form `ROLE:entity`. See [GCS Object ACL documentation](https://cloud.google.com/storage/docs/json_api/v1/objectAccessControls) for more details. Must be set if `predefined_acl` is not.

## Attributes Reference

Only the arguments listed above are exposed as attributes.
