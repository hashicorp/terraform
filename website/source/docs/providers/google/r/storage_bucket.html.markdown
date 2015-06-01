---
layout: "google"
page_title: "Google: google_storage_bucket"
sidebar_current: "docs-google-resource-storage"
description: |-
  Creates a new bucket in Google Cloud Storage.
---

# google\_storage\_bucket

Creates a new bucket in Google cloud storage service(GCS). Currently, it will not change location nor ACL once a bucket has been created with Terraform. For more information see [the official documentation](https://cloud.google.com/storage/docs/overview) and [API](https://cloud.google.com/storage/docs/json_api).


## Example Usage

Example creating a private bucket in standard storage, in the EU region. 

```
resource "google_storage_bucket" "image-store" {
	name = "image-store-bucket"
	predefined_acl = "projectPrivate"
	location = "EU"
}

```

## Argument Reference

* `name` - (Required) The name of the bucket.
* `predefined_acl` - (Optional, Default: 'private') The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control#predefined-acl) to apply.
* `location` - (Optional, Default: 'US') The [GCS location](https://cloud.google.com/storage/docs/bucket-locations) 
* `force_destroy` - (Optional, Default: false) When deleting a bucket, this boolean option will delete all contained objects. If you try to delete a bucket that contains objects, Terraform will fail that run. 
