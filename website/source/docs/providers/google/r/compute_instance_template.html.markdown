---
layout: "google"
page_title: "Google: google_compute_instance_template"
sidebar_current: "docs-google-compute-instance-template"
description: |-
  Manages a VM instance template resource within GCE.
---


# google\_compute\_instance\_template

Manages a VM instance template resource within GCE. For more information see
[the official documentation](https://cloud.google.com/compute/docs/instance-templates)
and
[API](https://cloud.google.com/compute/docs/reference/latest/instanceTemplates).


## Example Usage

```js
resource "google_compute_instance_template" "foobar" {
  name        = "terraform-test"
  description = "template description"

  tags = ["foo", "bar"]

  instance_description = "description assigned to instances"
  machine_type         = "n1-standard-1"
  can_ip_forward       = false
  automatic_restart    = true
  on_host_maintenance  = "MIGRATE"

  // Create a new boot disk from an image
  disk {
    source_image = "debian-7-wheezy-v20160301"
    auto_delete = true
    boot = true
  }

  // Use an existing disk resource
  disk {
    source      = "foo_existing_disk"
    auto_delete = false
    boot        = false
  }

  network_interface {
    network = "default"
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

* `disk` - (Required) Disks to attach to instances created from this template.
    This can be specified multiple times for multiple disks. Structure is
    documented below.

* `machine_type` - (Required) The machine type to create.

* `name` - (Required) A unique name for the resource, required by GCE.

- - -

* `can_ip_forward` - (Optional) Whether to allow sending and receiving of
    packets with non-matching source or destination IPs. This defaults to false.

* `description` - (Optional) A brief description of this resource.

* `instance_description` - (Optional) A brief description to use for instances
    created from this template.

* `metadata` - (Optional) Metadata key/value pairs to make available from
    within instances created from this template.

* `network_interface` - (Required) Networks to attach to instances created from
    this template. This can be specified multiple times for multiple networks.
    Structure is documented below.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `region` - (Optional) An instance template is a global resource that is not
    bound to a zone or a region. However, you can still specify some regional
    resources in an instance template, which restricts the template to the
    region where that resource resides. For example, a custom `subnetwork`
    resource is tied to a specific region. Defaults to the region of the
    Provider if no value is given.

* `scheduling` - (Optional) The scheduling strategy to use. More details about
    this configuration option are detailed below.

* `service_account` - (Optional) Service account to attach to the instance.

* `tags` - (Optional) Tags to attach to the instance.

The `disk` block supports:

* `auto_delete` - (Optional) Whether or not the disk should be auto-deleted.
    This defaults to true.

* `boot` - (Optional) Indicates that this is a boot disk.

* `device_name` - (Optional) A unique device name that is reflected into the
    /dev/  tree of a Linux operating system running within the instance. If not
    specified, the server chooses a default device name to apply to this disk.

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

* `disk_type` - (Optional) The GCE disk type. Can be either `"pd-ssd"`,
    `"local-ssd"`, or `"pd-standard"`.

* `disk_size_gb` - (Optional) The size of the image in gigabytes. If not
    specified, it will inherit the size of its base image.

* `type` - (Optional) The type of GCE disk, can be either `"SCRATCH"` or
    `"PERSISTENT"`.

The `network_interface` block supports:

* `network` - (Optional) The name of the network to attach this interface to.
    Use `network` attribute for Legacy or Auto subnetted networks and
    `subnetwork` for custom subnetted networks.

* `subnetwork` - (Optional) the name of the subnetwork to attach this interface
    to. The subnetwork must exist in the same `region` this instance will be
    created in. Either `network` or `subnetwork` must be provided.

* `access_config` - (Optional) Access configurations, i.e. IPs via which this
    instance can be accessed via the Internet. Omit to ensure that the instance
    is not accessible from the Internet (this means that ssh provisioners will
    not work unless you are running Terraform can send traffic to the instance's
    network (e.g. via tunnel or because it is running on another cloud instance
    on that network). This block can be repeated multiple times. Structure documented below.

The `access_config` block supports:

* `nat_ip` - (Optional) The IP address that will be 1:1 mapped to the instance's
    network ip. If not given, one will be generated.

The `service_account` block supports:

* `scopes` - (Required) A list of service scopes. Both OAuth2 URLs and gcloud
    short names are supported.

The `scheduling` block supports:

* `automatic_restart` - (Optional) Specifies whether the instance should be
    automatically restarted if it is terminated by Compute Engine (not
    terminated by a user). This defaults to true.

* `on_host_maintenance` - (Optional) Defines the maintenance behavior for this
    instance.

* `preemptible` - (Optional) Allows instance to be preempted. Read more on this
    [here](https://cloud.google.com/compute/docs/instances/preemptible).

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `metadata_fingerprint` - The unique fingerprint of the metadata.

* `self_link` - The URI of the created resource.

* `tags_fingerprint` - The unique fingerprint of the tags.
