---
layout: "openstack"
page_title: "OpenStack: openstack_blockstorage_volume_attach_v2"
sidebar_current: "docs-openstack-resource-blockstorage-volume-attach-v2"
description: |-
  Attaches a Block Storage Volume to an Instance.
---

# openstack\_blockstorage\_volume_attach_v2

Attaches a Block Storage Volume to an Instance using the OpenStack
Block Storage (Cinder) v2 API.

## Example Usage

```
resource "openstack_blockstorage_volume_v2" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
}

resource "openstack_blockstorage_volume_attach_v2" "va_1" {
  instance_id = "${openstack_compute_instance_v2.instance_1.id}"
  volume_id = "${openstack_blockstorage_volume_v2.volume_1.id}"
  device = "auto"
  attach_mode = "rw"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Block Storage
    client. A Block Storage client is needed to create a volume attachment.
    If omitted, the `OS_REGION_NAME` environment variable is used. Changing
    this creates a new volume attachment.

* `volume_id` - (Required) The ID of the Volume to attach to an Instance.

* `instance_id` - (Required if `host_name` is not used) The ID of the Instance
  to attach the Volume to.

* `host_name` - (Required if `instance_id` is not used) The host to attach the
  volume to.

* `device` - (Optional) The device to attach the volume as.

* `attach_mode` - (Optional) Specify whether to attach the volume as Read-Only
  (`ro`) or Read-Write (`rw`). Only values of `ro` and `rw` are accepted.
  If left unspecified, the Block Storage API will apply a default of `rw`.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `volume_id` - See Argument Reference above.
* `instance_id` - See Argument Reference above.
* `host_name` - See Argument Reference above.
* `attach_mode` - See Argument Reference above.
* `device` - See Argument Reference above.
  _NOTE_: Whether or not this is really the device the volume was attached
  as depends on the hypervisor being used in the OpenStack cloud. Do not
  consider this an authoritative piece of information.

## Import

Volume Attachments can be imported using the Volume and Attachment ID
separated by a slash, e.g.

```
$ terraform import openstack_blockstorage_volume_attach_v2.va_1 89c60255-9bd6-460c-822a-e2b959ede9d2/45670584-225f-46c3-b33e-6707b589b666
```

