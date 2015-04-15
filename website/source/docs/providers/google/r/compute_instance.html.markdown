---
layout: "google"
page_title: "Google: google_compute_instance"
sidebar_current: "docs-google-resource-instance"
description: |-
  Manages a VM instance resource within GCE.
---

# google\_compute\_instance

Manages a VM instance resource within GCE.  For more information see
[the official documentation](https://cloud.google.com/compute/docs/instances)
and
[API](https://cloud.google.com/compute/docs/reference/latest/instances).


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

	network_interface {
		network = "default"
        access_config {
            // Ephemeral IP
        }
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

* `network_interface` - (Required) Networks to attach to the instance. This can be
    specified multiple times for multiple networks. Structure is documented
    below.

* `network` - (DEPRECATED, Required) Networks to attach to the instance. This can be
    specified multiple times for multiple networks. Structure is documented
    below.

* `service_account` - (Optional) Service account to attach to the instance.

* `tags` - (Optional) Tags to attach to the instance.

The `disk` block supports:

* `disk` - (Required if image not set) The name of the disk (such as
     those managed by `google_compute_disk`) to attach.

* `image` - (Required if disk not set) The image from which to initialize this
  disk.  Either the full URL, a contraction of the form "project/name", or just
  a name (in which case the current project is used).

* `auto_delete` - (Optional) Whether or not the disk should be auto-deleted.
    This defaults to true.

* `type` - (Optional) The GCE disk type.

* `size` - (Optional) The size of the image in gigabytes. If not specified,
    it will inherit the size of its base image.

The `network_interface` block supports:

* `network` - (Required) The name of the network to attach this interface to.

* `access_config` - (Optional) Access configurations, i.e. IPs via which this instance can be
  accessed via the Internet.  Omit to ensure that the instance is not accessible from the Internet
(this means that ssh provisioners will not work unless you are running Terraform can send traffic to
the instance's network (e.g. via tunnel or because it is running on another cloud instance on that
network).  This block can be repeated multiple times.  Structure documented below.

The `access_config` block supports:

* `nat_ip` - (Optional) The IP address that will be 1:1 mapped to the instance's network ip.  If not
  given, one will be generated.

(DEPRECATED) The `network` block supports:

* `source` - (Required) The name of the network to attach this interface to.

* `address` - (Optional) The IP address of a reserved IP address to assign
     to this interface.

The `service_account` block supports:

* `scopes` - (Required) A list of service scopes. Both OAuth2 URLs and gcloud
    short names are supported.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `machine_type` - The type of machine.
* `zone` - The zone the machine lives in.
