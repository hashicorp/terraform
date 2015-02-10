---
layout: "google"
page_title: "Google: google_compute_instance_template"
sidebar_current: "docs-google-resource-instance_template"
description: |-
  Manages a VM instance template resource within GCE.
---


# google\_compute\_instance\_template

Manages a VM instance template resource within GCE.  For more information see
[the official documentation](https://cloud.google.com/compute/docs/instance-templates)
and
[API](https://cloud.google.com/compute/docs/reference/latest/instanceTemplates).


## Example Usage

```
resource "google_compute_instance_template" "foobar" {
	name = "terraform-test"
	description = "template description"
	instance_description = "description assigned to instances"
	machine_type = "n1-standard-1"
	can_ip_forward = false
	automatic_restart = true
	on_host_maintenance = "MIGRATE"
	tags = ["foo", "bar"]

	# Create a new boot disk from an image
	disk {
		source_image = "debian-7-wheezy-v20140814"
		auto_delete = true
		boot = true
	}

	# Use an existing disk resource
	disk {
		source = "foo_existing_disk"
		auto_delete = false
		boot = false
	}

	network {
		source = "default"
	}

	metadata {
		foo = "bar"
	}

	service_account {
		scopes = ["userinfo-email", "compute-ro", "storage-ro"]
	}
}
```

## Argument Reference

Note that changing any field for this resource forces a new resource to be created.

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.

* `description` - (Optional) A brief description of this resource.

* `can_ip_forward` - (Optional) Whether to allow sending and receiving of
	packets with non-matching source or destination IPs.
	This defaults to false.

* `instance_description` - (Optional) A brief description to use for instances
	created from this template.

* `machine_type` - (Required) The machine type to create.

* `disk` - (Required) Disks to attach to instances created from this
	template. This can be specified multiple times for multiple disks.
	Structure is documented below.

* `metadata` - (Optional) Metadata key/value pairs to make available from
	within instances created from this template.

* `network` - (Required) Networks to attach to instances created from this template.
 	This can be specified multiple times for multiple networks. Structure is
	documented below.

* `automatic_restart` - (Optional) Specifies whether the instance should be
	automatically restarted if it is terminated by Compute Engine (not
	terminated by a user).
	This defaults to true.

* `on_host_maintenance` - (Optional) Defines the maintenance behavior for this
	instance.

* `service_account` - (Optional) Service account to attach to the instance.

* `tags` - (Optional) Tags to attach to the instance.



The `disk` block supports:

* `auto_delete` - (Optional) Whether or not the disk should be auto-deleted.
	This defaults to true.

* `boot` - (Optional) Indicates that this is a boot disk.

* `device_name` - (Optional) A unique device name that is reflected into
	the /dev/  tree of a Linux operating system running within the instance.
	If not specified, the server chooses a default device name to apply to
	this disk.

* `disk_name` - (Optional) Name of the disk. When not provided, this defaults
	to the name of the instance.

* `source_image` - (Required if source not set) The name of the image to base
	this disk off of.

* `interface` - (Optional) Specifies the disk interface to use for attaching
	this disk.

* `mode` - (Optional) The mode in which to attach this disk, either READ_WRITE
	or READ_ONLY. If you are attaching or creating a boot disk, this must
	read-write mode.

* `source` - (Required if source_image not set) The name of the disk (such as
	those managed by `google_compute_disk`) to attach.

* `type` - (Optional) The GCE disk type.

The `network` block supports:

* `source` - (Required) The name of the network to attach this interface to.

* `address` - (Optional) The IP address of a reserved IP address to assign
	to this interface.

The `service_account` block supports:

* `scopes` - (Required) A list of service scopes. Both OAuth2 URLs and gcloud
	short names are supported.

## Attributes Reference

The following attributes are exported:

* `self_link` - The URL of the created resource.
