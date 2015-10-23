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

```
resource "google_compute_firewall" "default" {
	name = "test"
	network = "${google_compute_network.other.name}"

	allow {
		protocol = "icmp"
	}

	allow {
		protocol = "tcp"
		ports = ["80", "8080", "1000-2000"]
	}

	source_tags = ["web"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `description` - (Optional) Textual description field.

* `network` - (Required) The name of the network to attach this firewall to.

* `allow` - (Required) Can be specified multiple times for each allow
    rule. Each allow block supports fields documented below.

* `source_ranges` - (Optional) A list of source CIDR ranges that this
   firewall applies to.

* `source_tags` - (Optional) A list of source tags that this firewall applies to.

* `target_tags` - (Optional) A list of target tags that this firewall applies to.

The `allow` block supports:

* `protocol` - (Required) The name of the protocol to allow.

* `ports` - (Optional) List of ports and/or port ranges to allow. This can
    only be specified if the protocol is TCP or UDP.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `network` - The network that this resource is attached to.
* `source_ranges` - The CIDR block ranges this firewall applies to.
* `source_tags` - The tags that this firewall applies to.
