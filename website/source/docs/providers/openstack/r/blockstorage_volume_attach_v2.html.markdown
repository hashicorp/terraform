---
layout: "openstack"
page_title: "OpenStack: openstack_blockstorage_volume_attach_v2"
sidebar_current: "docs-openstack-resource-blockstorage-volume-attach-v2"
description: |-
  Creates an attachment connection to a Block Storage volume
---

# openstack\_blockstorage\_volume\_attach\_v2

This resource is experimental and may be removed in the future! Feedback
is requested if you find this resource useful or if you find any problems
with it.

Creates a general purpose attachment connection to a Block
Storage volume using the OpenStack Block Storage (Cinder) v2 API.
Depending on your Block Storage service configuration, this
resource can assist in attaching a volume to a non-OpenStack resource
such as a bare-metal server or a remote virtual machine in a
different cloud provider.

This does not actually attach a volume to an instance. Please use
the `openstack_compute_volume_attach_v2` resource for that.

## Example Usage

```hcl
resource "openstack_blockstorage_volume_v2" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_blockstorage_volume_attach_v2" "va_1" {
  volume_id = "${openstack_blockstorage_volume_v2.volume_1.id}"
  device = "auto"
  host_name = "devstack"
  ip_address = "192.168.255.10"
  initiator = "iqn.1993-08.org.debian:01:e9861fb1859"
  os_type = "linux2"
  platform = "x86_64"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Block Storage
    client. A Block Storage client is needed to create a volume attachment.
    If omitted, the `OS_REGION_NAME` environment variable is used. Changing
    this creates a new volume attachment.

* `attach_mode` - (Optional) Specify whether to attach the volume as Read-Only
  (`ro`) or Read-Write (`rw`). Only values of `ro` and `rw` are accepted.
  If left unspecified, the Block Storage API will apply a default of `rw`.

* `device` - (Optional) The device to tell the Block Storage service this
  volume will be attached as. This is purely for informational purposes.
  You can specify `auto` or a device such as `/dev/vdc`.

* `host_name` - (Required) The host to attach the volume to.

* `initiator` - (Optional) The iSCSI initiator string to make the connection.

* `ip_address` - (Optional) The IP address of the `host_name` above.

* `multipath` - (Optional) Whether to connect to this volume via multipath.

* `os_type` - (Optional) The iSCSI initiator OS type.

* `platform` - (Optional) The iSCSI initiator platform.

* `volume_id` - (Required) The ID of the Volume to attach to an Instance.

* `wwpn` - (Optional) An array of wwpn strings. Used for Fibre Channel
  connections.

* `wwnn` - (Optional) A wwnn name. Used for Fibre Channel connections.

## Attributes Reference

In addition to the above, the following attributes are exported:

* `data` - This is a map of key/value pairs that contain the connection
  information. You will want to pass this information to a provisioner
  script to finalize the connection. See below for more information.

* `driver_volume_type` - The storage driver that the volume is based on.

* `mount_point_base` - A mount point base name for shared storage.

## Volume Connection Data

Upon creation of this resource, a `data` exported attribute will be available.
This attribute is a set of key/value pairs that contains the information
required to complete the block storage connection.

As an example, creating an iSCSI-based volume will return the following:

```
data.access_mode = rw
data.auth_method = CHAP
data.auth_password = xUhbGKQ8QCwKmHQ2
data.auth_username = Sphn5X4EoyFUUMYVYSA4
data.target_iqn = iqn.2010-10.org.openstack:volume-2d87ed25-c312-4f42-be1d-3b36b014561d
data.target_portal = 192.168.255.10:3260
data.volume_id = 2d87ed25-c312-4f42-be1d-3b36b014561d
```

This information can then be fed into a provisioner or a template shell script,
where the final result would look something like:

```
iscsiadm -m node -T ${self.data.target_iqn} -p ${self.data.target_portal} --interface default --op new
iscsiadm -m node -T ${self.data.target_iqn} -p ${self.data.target_portal} --op update -n node.session.auth.authmethod -v ${self.data.auth_method}
iscsiadm -m node -T ${self.data.target_iqn} -p ${self.data.target_portal} --op update -n node.session.auth.username -v ${self.data.auth_username}
iscsiadm -m node -T ${self.data.target_iqn} -p ${self.data.target_portal} --op update -n node.session.auth.password -v ${self.data.auth_password}
iscsiadm -m node -T ${self.data.target_iqn} -p ${self.data.target_portal} --login
iscsiadm -m node -T ${self.data.target_iqn} -p ${self.data.target_portal} --op update -n node.startup -v automatic
iscsiadm -m node -T ${self.data.target_iqn} -p ${self.data.target_portal} --rescan
```

The contents of `data` will vary from each Block Storage service. You must have
a good understanding of how the service is configured and how to make the
appropriate final connection. However, if used correctly, this has the
flexibility to be able to attach OpenStack Block Storage volumes to
non-OpenStack resources.

## Import

It is not possible to import this resource.
