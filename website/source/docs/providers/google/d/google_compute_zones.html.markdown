---
layout: "google"
page_title: "Google: google_compute_zones"
sidebar_current: "docs-google-datasource-compute-zones"
description: |-
  Provides a list of available Google Compute zones
---

# google\_compute\_zones

Provides access to available Google Compute zones in a region for a given project.
See more about [regions and zones](https://cloud.google.com/compute/docs/regions-zones/regions-zones) in the upstream docs.

```
data "google_compute_zones" "available" {}

resource "google_compute_instance_group_manager" "foo" {
  count = "${length(data.google_compute_zones.available.names)}"

  name               = "terraform-test-${count.index}"
  instance_template  = "${google_compute_instance_template.foobar.self_link}"
  base_instance_name = "foobar-${count.index}"
  zone               = "${data.google_compute_zones.available.names[count.index]}"
  target_size        = 1
}
```

## Argument Reference

The following arguments are supported:

* `region` (Optional) - Region from which to list available zones. Defaults to region declared in the provider.
* `status` (Optional) - Allows to filter list of zones based on their current status. Status can be either `UP` or `DOWN`.
  Defaults to no filtering (all available zones - both `UP` and `DOWN`).

## Attributes Reference

The following attribute is exported:

* `names` - A list of zones available in the given region
