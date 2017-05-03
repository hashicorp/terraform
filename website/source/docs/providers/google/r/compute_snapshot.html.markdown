---
layout: "google"
page_title: "Google: google_compute_snapshot"
sidebar_current: "docs-google-compute-snapshot"
description: |-
  Creates a new snapshot of a disk within GCE.
---

# google\_compute\_snapshot

Creates a new snapshot of a disk within GCE.

## Example Usage

```js
resource "google_compute_snapshot" "default" {
  name  = "test-snapshot"
  source_disk  = "test-disk"
  zone  = "us-central1-a"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `zone` - (Required) The zone where the source disk is located.

* `source_disk` - (Required) The disk which will be used as the source of the snapshot.

- - -

* `source_disk_encryption_key_raw` - (Optional) A 256-bit [customer-supplied encryption key]
    (https://cloud.google.com/compute/docs/disks/customer-supplied-encryption),
    encoded in [RFC 4648 base64](https://tools.ietf.org/html/rfc4648#section-4)
    to decrypt the source disk.

* `snapshot_encryption_key_raw` - (Optional) A 256-bit [customer-supplied encryption key]
    (https://cloud.google.com/compute/docs/disks/customer-supplied-encryption),
    encoded in [RFC 4648 base64](https://tools.ietf.org/html/rfc4648#section-4)
    to encrypt this snapshot.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `snapshot_encryption_key_sha256` - The [RFC 4648 base64]
    (https://tools.ietf.org/html/rfc4648#section-4) encoded SHA-256 hash of the
    [customer-supplied encryption key](https://cloud.google.com/compute/docs/disks/customer-supplied-encryption)
    that protects this resource.

* `source_disk_encryption_key_sha256` - The [RFC 4648 base64]
    (https://tools.ietf.org/html/rfc4648#section-4) encoded SHA-256 hash of the
    [customer-supplied encryption key](https://cloud.google.com/compute/docs/disks/customer-supplied-encryption)
    that protects the source disk.

* `source_disk_link` - The URI of the source disk.

* `self_link` - The URI of the created resource.
