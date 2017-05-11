---
layout: "google"
page_title: "Google: google_compute_backend_bucket"
sidebar_current: "docs-google-compute-backend-bucket"
description: |-
  Creates a Backend Bucket resource for Google Compute Engine.
---

# google\_compute\_backend\_bucket

A Backend Bucket defines a Google Cloud Storage bucket that will serve traffic through Google Cloud
Load Balancer.

## Example Usage

```hcl
resource "google_compute_backend_bucket" "foobar" {
  name        = "image-backend-bucket"
  description = "Contains beautiful images"
  bucket_name = "${google_storage_bucket.image_bucket.name}"
  enable_cdn  = true
}

resource "google_storage_bucket" "image_bucket" {
  name     = "image-store-bucket"
  location = "EU"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the backend bucket.

* `bucket_name` - (Required) The name of the Google Cloud Storage bucket to be used as a backend
    bucket.

- - -

* `description` - (Optional) The textual description for the backend bucket.

* `enable_cdn` - (Optional) Whether or not to enable the Cloud CDN on the backend bucket.

* `project` - (Optional) The project in which the resource belongs. If it is not provided, the
    provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are exported:

* `self_link` - The URI of the created resource.
