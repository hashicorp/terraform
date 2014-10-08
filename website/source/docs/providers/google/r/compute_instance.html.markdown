---
layout: "google"
page_title: "Google: google_compute_instance"
sidebar_current: "docs-google-resource-instance"
---

# google\_compute\_instance

Manages a VM instance resource within GCE.

## Example Usage

```
resource "google_compute_instance" "default" {
	name = "test"
	machine_type = "n1-standard-1"
	zone = "us-central1-a"
	tags = ["foo", "bar"]

	disk {
		image = "debian-7-wheezy-v20140814"
	}

	network {
		source = "default"
	}

	metadata {
		foo = "bar"
	}
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `description` - (Optional) A brief description of this resource.

* `machine_type` - (Required) The machine type to create.

* `zone` - (Required) The zone that the machine should be created in.

* `disk` - (Required) Disks to attach to the instance. This can be specified
    multiple times for multiple disks. Structure is documented below.

* `can_ip_forward` - (Optional) Whether to allow sending and receiving of
    packets with non-matching source or destination IPs.
    This defaults to false.

* `metadata` - (Optional) Metadata key/value pairs to make available from
    within the instance.

* `network` - (Required) Networks to attach to the instance. This can be
    specified multiple times for multiple networks. Structure is documented
    below.

* `tags` - (Optional) Tags to attach to the instance.

The `disk` block supports:

* `disk` - (Required if image not set) The name of the disk (such as
     those managed by `google_compute_disk`) to attach.

* `image` - (Required if disk not set) The name of the image to base
    this disk off of.

* `auto_delete` - (Optional) Whether or not the disk should be auto-deleted.
    This defaults to true.

* `type` - (Optional) The GCE disk type.

The `network` block supports:

* `source` - (Required) The name of the network to attach this interface to.

* `address` - (Optional) The IP address of a reserved IP address to assign
     to this interface.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `machine_type` - The type of machine.
* `zone` - The zone the machine lives in.
