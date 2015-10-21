---
layout: "google"
page_title: "Google: google_storage_bucket"
sidebar_current: "docs-google-storage-bucket"
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
	location = "EU"
    website {
        main_page_suffix = "index.html"
        not_found_page = "404.html"
    }
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the bucket.
* `predefined_acl` - (Optional, Deprecated) The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control#predefined-acl) to apply. Please switch
to `google_storage_bucket_acl.predefined_acl`.
* `location` - (Optional, Default: 'US') The [GCS location](https://cloud.google.com/storage/docs/bucket-locations) 
* `force_destroy` - (Optional, Default: false) When deleting a bucket, this boolean option will delete all contained objects. If you try to delete a bucket that contains objects, Terraform will fail that run. 

The optional `website` block supports:

* `main_page_suffix` - (Optional) Behaves as the bucket's directory index where missing objects are treated as potential directories.
* `not_found_page` - (Optional) The custom object to return when a requested resource is not found.

## Attributes Reference

The following attributes are exported:

* `self_link` - The URI of the created resource.
