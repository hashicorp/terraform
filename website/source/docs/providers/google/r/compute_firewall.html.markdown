---
layout: "google"
page_title: "Google: google_compute_firewall"
sidebar_current: "docs-google-compute-firewall"
description: |-
  Manages a firewall resource within GCE.
---

# google\_compute\_firewall

Manages a firewall resource within GCE.

## Example Usage

```hcl
resource "google_compute_firewall" "default" {
  name    = "test"
  network = "${google_compute_network.other.name}"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["80", "8080", "1000-2000"]
  }

  source_tags = ["web"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `network` - (Required) The name of the network to attach this firewall to.

* `allow` - (Required) Can be specified multiple times for each allow
    rule. Each allow block supports fields documented below.

- - -

* `description` - (Optional) Textual description field.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `source_ranges` - (Optional) A list of source CIDR ranges that this
   firewall applies to.

* `source_tags` - (Optional) A list of source tags for this firewall.

* `target_tags` - (Optional) A list of target tags for this firewall.

The `allow` block supports:

* `protocol` - (Required) The name of the protocol to allow.

* `ports` - (Optional) List of ports and/or port ranges to allow. This can
    only be specified if the protocol is TCP or UDP.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.
