---
layout: "openstack"
page_title: "OpenStack: openstack_compute_instance_v2"
sidebar_current: "docs-openstack-resource-compute-instance-v2"
description: |-
  Manages a V2 VM instance resource within OpenStack.
---

# openstack\_compute\_instance_v2

Manages a V2 VM instance resource within OpenStack.

## Example Usage

```
resource "openstack_compute_instance_v2" "test-server" {
  name = "tf-test"
  image_id = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id = "3"
  metadata {
    this = "that"
  }
  key_pair = "my_key_pair_name"
  security_groups = ["test-group-1"]
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to create the server instance. If
    omitted, the `OS_REGION_NAME` environment variable is used. Changing this
    creates a new server.

* `name` - (Required) A unique name for the resource.

* `image_id` - (Optional; Required if `image_name` is empty and not booting
    from a volume) The image ID of the desired image for the server. Changing
    this creates a new server.

* `image_name` - (Optional; Required if `image_id` is empty and not booting
    from a volume) The name of the desired image for the server. Changing this
    creates a new server.

* `flavor_id` - (Optional; Required if `flavor_name` is empty) The flavor ID of
    the desired flavor for the server. Changing this resizes the existing server.

* `flavor_name` - (Optional; Required if `flavor_id` is empty) The name of the
    desired flavor for the server. Changing this resizes the existing server.

* `floating_ip` - (Optional) A *Compute* Floating IP that will be associated
    with the Instance. The Floating IP must be provisioned already.

* `user_data` - (Optional) The user data to provide when launching the instance.
    Changing this creates a new server.

* `security_groups` - (Optional) An array of one or more security group names
    to associate with the server. Changing this results in adding/removing
    security groups from the existing server.

* `availability_zone` - (Optional) The availability zone in which to create
    the server. Changing this creates a new server.

* `network` - (Optional) An array of one or more networks to attach to the
    instance. The network object structure is documented below. Changing this
    creates a new server.

* `metadata` - (Optional) Metadata key/value pairs to make available from
    within the instance. Changing this updates the existing server metadata.

* `config_drive` - (Optional) Whether to use the config_drive feature to
    configure the instance. Changing this creates a new server.

* `admin_pass` - (Optional) The administrative password to assign to the server.
    Changing this changes the root password on the existing server.

* `key_pair` - (Optional) The name of a key pair to put on the server. The key
    pair must already be created and associated with the tenant's account.
    Changing this creates a new server.

* `block_device` - (Optional) The object for booting by volume. The block_device
    object structure is documented below. Changing this creates a new server.

* `volume` - (Optional) Attach an existing volume to the instance. The volume
    structure is described below.

* `scheduler_hints` - (Optional) Provider the Nova scheduler with hints on how
    the instance should be launched. The available hints are described below.

The `network` block supports:

* `uuid` - (Required unless `port`  or `name` is provided) The network UUID to
    attach to the server.

* `name` - (Required unless `uuid` or `port` is provided) The human-readable
    name of the network.

* `port` - (Required unless `uuid` or `name` is provided) The port UUID of a
    network to attach to the server.

* `fixed_ip_v4` - (Optional) Specifies a fixed IPv4 address to be used on this
    network.

The `block_device` block supports:

* `uuid` - (Required) The UUID of the image, volume, or snapshot.

* `source_type` - (Required) The source type of the device. Must be one of
    "image", "volume", or "snapshot".

* `volume_size` - (Optional) The size of the volume to create (in gigabytes).

* `boot_index` - (Optional) The boot index of the volume. It defaults to 0.

* `destination_type` - (Optional) The type that gets created. Possible values
    are "volume" and "local".

The `volume` block supports:

* `volume_id` - (Required) The UUID of the volume to attach.

* `device` - (Optional) The device that the volume will be attached as. For
    example:  `/dev/vdc`. Omit this option to allow the volume to be
    auto-assigned a device.

The `scheduler_hints` block supports:

* `group` - (Optional) A UUID of a Server Group. The instance will be placed
    into that group.

* `different_host` - (Optional) A list of instance UUIDs. The instance will
    be scheduled on a different host than all other instances.

* `same_host` - (Optional) A list of instance UUIDs. The instance will be
    scheduled on the same host of those specified.

* `query` - (Optional) A conditional query that a compute node must pass in
    order to host an instance.

* `target_cell` - (Optional) The name of a cell to host the instance.

* `build_near_host_ip` - (Optional) An IP Address in CIDR form. The instance
    will be placed on a compute node that is in the same subnet.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `access_ip_v4` - The first detected Fixed IPv4 address _or_ the
    Floating IP.
* `access_ip_v6` - The first detected Fixed IPv6 address.
* `metadata` - See Argument Reference above.
* `security_groups` - See Argument Reference above.
* `flavor_id` - See Argument Reference above.
* `flavor_name` - See Argument Reference above.
* `network/uuid` - See Argument Reference above.
* `network/name` - See Argument Reference above.
* `network/port` - See Argument Reference above.
* `network/fixed_ip_v4` - The Fixed IPv4 address of the Instance on that
    network.
* `network/fixed_ip_v6` - The Fixed IPv6 address of the Instance on that
    network.
* `network/mac` - The MAC address of the NIC on that network.

## Notes

If you configure the instance to have multiple networks, be aware that only
the first network can be associated with a Floating IP. So the first network
in the instance resource _must_ be the network that you have configured to
communicate with your floating IP / public network via a Neutron Router.
