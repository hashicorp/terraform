---
layout: "google"
page_title: "Google: google_storage_bucket"
sidebar_current: "docs-google-storage-bucket"
description: |-
  Creates a new bucket in Google Cloud Storage.
---

# google\_storage\_bucket

Creates a new bucket in Google cloud storage service (GCS). 
Once a bucket has been created, its location can't be changed.
[ACLs](https://cloud.google.com/storage/docs/access-control/lists) can be applied using the `google_storage_bucket_acl` resource.
For more information see 
[the official documentation](https://cloud.google.com/storage/docs/overview) 
and 
[API](https://cloud.google.com/storage/docs/json_api/v1/buckets).


## Example Usage

Example creating a private bucket in standard storage, in the EU region.

```hcl
resource "google_storage_bucket" "image-store" {
  name     = "image-store-bucket"
  location = "EU"

  website {
    main_page_suffix = "index.html"
    not_found_page   = "404.html"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the bucket.

- - -

* `force_destroy` - (Optional, Default: false) When deleting a bucket, this
    boolean option will delete all contained objects. If you try to delete a
    bucket that contains objects, Terraform will fail that run.

* `location` - (Optional, Default: 'US') The [GCS location](https://cloud.google.com/storage/docs/bucket-locations)


* `predefined_acl` - (Optional, Deprecated) The [canned GCS ACL](https://cloud.google.com/storage/docs/access-control#predefined-acl) to apply. Please switch
to `google_storage_bucket_acl.predefined_acl`.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `storage_class` - (Optional) The [Storage Class](https://cloud.google.com/storage/docs/storage-classes) of the new bucket. Supported values include: `MULTI_REGIONAL`, `REGIONAL`, `NEARLINE`, `COLDLINE`.

* `website` - (Optional) Configuration if the bucket acts as a website. Structure is documented below.

* `cors` - (Optional) The bucket's [Cross-Origin Resource Sharing (CORS)](https://www.w3.org/TR/cors/) configuration. Multiple blocks of this type are permitted. Structure is documented below.

The `website` block supports:

* `main_page_suffix` - (Optional) Behaves as the bucket's directory index where
    missing objects are treated as potential directories.

* `not_found_page` - (Optional) The custom object to return when a requested
    resource is not found.
    
The `cors` block supports:

* `origin` - (Optional) The list of [Origins](https://tools.ietf.org/html/rfc6454) eligible to receive CORS response headers. Note: "*" is permitted in the list of origins, and means "any Origin".
    
* `method` - (Optional) The list of HTTP methods on which to include CORS response headers, (GET, OPTIONS, POST, etc) Note: "*" is permitted in the list of methods, and means "any method".
    
* `response_header` - (Optional) The list of HTTP headers other than the [simple response headers](https://www.w3.org/TR/cors/#simple-response-header) to give permission for the user-agent to share across domains.
    
* `max_age_seconds` - (Optional) The value, in seconds, to return in the [Access-Control-Max-Age header](https://www.w3.org/TR/cors/#access-control-max-age-response-header) used in preflight responses.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.

* `url` - The base URL of the bucket, in the format `gs://<bucket-name>`.
