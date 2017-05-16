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

### Basic Instance

```hcl
resource "openstack_compute_instance_v2" "basic" {
  name            = "basic"
  image_id        = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  metadata {
    this = "that"
  }

  network {
    name = "my_network"
  }
}
```

### Instance With Attached Volume

```hcl
resource "openstack_blockstorage_volume_v2" "myvol" {
  name = "myvol"
  size = 1
}

resource "openstack_compute_instance_v2" "myinstance" {
  name            = "myinstance"
  image_id        = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  network {
    name = "my_network"
  }
}

resource "openstack_compute_volume_attach_v2" "attached" {
  compute_id = "${openstack_compute_instance_v2.myinstance.id}"
  volume_id = "${openstack_blockstorage_volume_v2.myvol.id}"
}
```

### Boot From Volume

```hcl
resource "openstack_compute_instance_v2" "boot-from-volume" {
  name            = "boot-from-volume"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  block_device {
    uuid                  = "<image-id>"
    source_type           = "image"
    volume_size           = 5
    boot_index            = 0
    destination_type      = "volume"
    delete_on_termination = true
  }

  network {
    name = "my_network"
  }
}
```

### Boot From an Existing Volume

```hcl
resource "openstack_blockstorage_volume_v1" "myvol" {
  name     = "myvol"
  size     = 5
  image_id = "<image-id>"
}

resource "openstack_compute_instance_v2" "boot-from-volume" {
  name            = "bootfromvolume"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  block_device {
    uuid                  = "${openstack_blockstorage_volume_v1.myvol.id}"
    source_type           = "volume"
    boot_index            = 0
    destination_type      = "volume"
    delete_on_termination = true
  }

  network {
    name = "my_network"
  }
}
```

### Boot Instance, Create Volume, and Attach Volume as a Block Device

```hcl
resource "openstack_compute_instance_v2" "instance_1" {
  name            = "instance_1"
  image_id        = "<image-id>"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  block_device {
    uuid                  = "<image-id>"
    source_type           = "image"
    destination_type      = "local"
    boot_index            = 0
    delete_on_termination = true
  }

  block_device {
    source_type           = "blank"
    destination_type      = "volume"
    volume_size           = 1
    boot_index            = 1
    delete_on_termination = true
  }
}
```

### Boot Instance and Attach Existing Volume as a Block Device

```hcl
resource "openstack_blockstorage_volume_v2" "volume_1" {
  name = "volume_1"
  size = 1
}

resource "openstack_compute_instance_v2" "instance_1" {
  name            = "instance_1"
  image_id        = "<image-id>"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  block_device {
    uuid                  = "<image-id>"
    source_type           = "image"
    destination_type      = "local"
    boot_index            = 0
    delete_on_termination = true
  }

  block_device {
    uuid                  = "${openstack_blockstorage_volume_v2.volume_1.id}"
    source_type           = "volume"
    destination_type      = "volume"
    boot_index            = 1
    delete_on_termination = true
  }
}
```

### Instance With Multiple Networks

```hcl
resource "openstack_networking_floatingip_v2" "myip" {
  pool = "my_pool"
}

resource "openstack_compute_instance_v2" "multi-net" {
  name            = "multi-net"
  image_id        = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  network {
    name = "my_first_network"
  }

  network {
    name = "my_second_network"
  }
}

resource "openstack_compute_floatingip_associate_v2" "myip" {
  floating_ip = "${openstack_networking_floatingip_v2.myip.address}"
  instance_id = "${openstack_compute_instance_v2.multi-net.id}"
  fixed_ip = "${openstack_compute_instance_v2.multi-net.network.1.fixed_ip_v4}"
}
```

### Instance With Personality

```hcl
resource "openstack_compute_instance_v2" "personality" {
  name            = "personality"
  image_id        = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  personality {
    file    = "/path/to/file/on/instance.txt"
    content = "contents of file"
  }

  network {
    name = "my_network"
  }
}
```

### Instance with Multiple Ephemeral Disks

```hcl
resource "openstack_compute_instance_v2" "multi-eph" {
  name            = "multi_eph"
  image_id        = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]

  block_device {
    boot_index            = 0
    delete_on_termination = true
    destination_type      = "local"
    source_type           = "image"
    uuid                  = "<image-id>"
  }

  block_device {
    boot_index            = -1
    delete_on_termination = true
    destination_type      = "local"
    source_type           = "blank"
    volume_size           = 1
  }

  block_device {
    boot_index            = -1
    delete_on_termination = true
    destination_type      = "local"
    source_type           = "blank"
    volume_size           = 1
  }
}
```

### Instance with User Data (cloud-init)

```hcl
resource "openstack_compute_instance_v2" "instance_1" {
  name            = "basic"
  image_id        = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["default"]
  user_data       = "#cloud-config\nhostname: instance_1.example.com\nfqdn: instance_1.example.com"

  network {
    name = "my_network"
  }
}
```

`user_data` can come from a variety of sources: inline, read in from the `file`
function, or the `template_cloudinit_config` resource.

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to create the server instance. If
    omitted, the `OS_REGION_NAME` environment variable is used. Changing this
    creates a new server.

* `name` - (Required) A unique name for the resource.

* `image_id` - (Optional; Required if `image_name` is empty and not booting
    from a volume. Do not specify if booting from a volume.) The image ID of
    the desired image for the server. Changing this creates a new server.

* `image_name` - (Optional; Required if `image_id` is empty and not booting
    from a volume. Do not specify if booting from a volume.) The name of the
    desired image for the server. Changing this creates a new server.

* `flavor_id` - (Optional; Required if `flavor_name` is empty) The flavor ID of
    the desired flavor for the server. Changing this resizes the existing server.

* `flavor_name` - (Optional; Required if `flavor_id` is empty) The name of the
    desired flavor for the server. Changing this resizes the existing server.

* `floating_ip` - (Deprecated) A *Compute* Floating IP that will be associated
    with the Instance. The Floating IP must be provisioned already. See *Notes*
    for more information about Floating IPs.

* `user_data` - (Optional) The user data to provide when launching the instance.
    Changing this creates a new server.

* `security_groups` - (Optional) An array of one or more security group names
    to associate with the server. Changing this results in adding/removing
    security groups from the existing server. *Note*: When attaching the
    instance to networks using Ports, place the security groups on the Port
    and not the instance.

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

* `block_device` - (Optional) Configuration of block devices. The block_device
    structure is documented below. Changing this creates a new server.
    You can specify multiple block devices which will create an instance with
    multiple disks. This configuration is very flexible, so please see the
    following [reference](http://docs.openstack.org/developer/nova/block_device_mapping.html)
    for more information.

* `volume` - (Deprecated) Attach an existing volume to the instance. The volume
    structure is described below. *Note*: This is no longer the recommended
    method of attaching a volume to an instance. Please see `block_device`
    (above) or the `openstack_compute_volume_attach_v2` and
    `openstack_blockstorage_volume_attach_v2` resources.

* `scheduler_hints` - (Optional) Provide the Nova scheduler with hints on how
    the instance should be launched. The available hints are described below.

* `personality` - (Optional) Customize the personality of an instance by
    defining one or more files and their contents. The personality structure
    is described below.

* `stop_before_destroy` - (Optional) Whether to try stop instance gracefully
    before destroying it, thus giving chance for guest OS daemons to stop correctly.
    If instance doesn't stop within timeout, it will be destroyed anyway.

* `force_delete` - (Optional) Whether to force the OpenStack instance to be
    forcefully deleted. This is useful for environments that have reclaim / soft
    deletion enabled.


The `network` block supports:

* `uuid` - (Required unless `port`  or `name` is provided) The network UUID to
    attach to the server. Changing this creates a new server.

* `name` - (Required unless `uuid` or `port` is provided) The human-readable
    name of the network. Changing this creates a new server.

* `port` - (Required unless `uuid` or `name` is provided) The port UUID of a
    network to attach to the server. Changing this creates a new server.

* `fixed_ip_v4` - (Optional) Specifies a fixed IPv4 address to be used on this
    network. Changing this creates a new server.

* `fixed_ip_v6` - (Optional) Specifies a fixed IPv6 address to be used on this
    network. Changing this creates a new server.

* `floating_ip` - (Deprecated) Specifies a floating IP address to be associated
    with this network. Cannot be combined with a top-level floating IP. See
    *Notes* for more information about Floating IPs.

* `access_network` - (Optional) Specifies if this network should be used for
    provisioning access. Accepts true or false. Defaults to false.

The `block_device` block supports:

* `uuid` - (Required unless `source_type` is set to `"blank"` ) The UUID of
    the image, volume, or snapshot. Changing this creates a new server.

* `source_type` - (Required) The source type of the device. Must be one of
    "blank", "image", "volume", or "snapshot". Changing this creates a new
    server.

* `volume_size` - The size of the volume to create (in gigabytes). Required
    in the following combinations: source=image and destination=volume,
    source=blank and destination=local, and source=blank and destination=volume.
    Changing this creates a new server.

* `boot_index` - (Optional) The boot index of the volume. It defaults to 0.
    Changing this creates a new server.

* `destination_type` - (Optional) The type that gets created. Possible values
    are "volume" and "local". Changing this creates a new server.

* `delete_on_termination` - (Optional) Delete the volume / block device upon
    termination of the instance. Defaults to false. Changing this creates a
    new server.

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

The `personality` block supports:

* `file` - (Required) The absolute path of the destination file.

* `contents` - (Required) The contents of the file. Limited to 255 bytes.

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
* `network/floating_ip` - The Floating IP address of the Instance on that
    network.
* `network/mac` - The MAC address of the NIC on that network.
* `all_metadata` - Contains all instance metadata, even metadata not set
    by Terraform.

## Notes

### Floating IPs

Specifying Floating IPs within the instance is now deprecated. Please use
either the `openstack_compute_floatingip_associate_v2` resource or attach
the floating IP to an `openstack_networking_port_v2` resource.

Floating IPs can be associated in one of two ways:

* You can specify a Floating IP address by using the top-level `floating_ip`
attribute. This floating IP will be associated with either the network defined
in the first `network` block or the default network if no `network` blocks are
defined.

* You can specify a Floating IP address by using the `floating_ip` attribute
defined in the `network` block. Each `network` block can have its own floating
IP address.

Only one of the above methods can be used.

### Multiple Ephemeral Disks

It's possible to specify multiple `block_device` entries to create an instance
with multiple ephemeral (local) disks. In order to create multiple ephemeral
disks, the sum of the total amount of ephemeral space must be less than or
equal to what the chosen flavor supports.

The following example shows how to create an instance with multiple ephemeral
disks:

```
resource "openstack_compute_instance_v2" "foo" {
  name            = "terraform-test"
  security_groups = ["default"]

  block_device {
    boot_index            = 0
    delete_on_termination = true
    destination_type      = "local"
    source_type           = "image"
    uuid                  = "<image uuid>"
  }

  block_device {
    boot_index            = -1
    delete_on_termination = true
    destination_type      = "local"
    source_type           = "blank"
    volume_size           = 1
  }

  block_device {
    boot_index            = -1
    delete_on_termination = true
    destination_type      = "local"
    source_type           = "blank"
    volume_size           = 1
  }
}
```

### Instances and Ports

Neutron Ports are a great feature and provide a lot of functionality. However,
there are some notes to be aware of when mixing Instances and Ports:

* When attaching an Instance to one or more networks using Ports, place the
security groups on the Port and not the Instance. If you place the security
groups on the Instance, the security groups will not be applied upon creation,
but they will be applied upon a refresh. This is a known OpenStack bug.

* Network IP information is not available within an instance for networks that
are attached with Ports. This is mostly due to the flexibility Neutron Ports
provide when it comes to IP addresses. For example, a Neutron Port can have
multiple Fixed IP addresses associated with it. It's not possible to know which
single IP address the user would want returned to the Instance's state
information. Therefore, in order for a Provisioner to connect to an Instance
via it's network Port, customize the `connection` information:

```hcl
resource "openstack_networking_port_v2" "port_1" {
  name           = "port_1"
  admin_state_up = "true"

  network_id = "0a1d0a27-cffa-4de3-92c5-9d3fd3f2e74d"

  security_group_ids = [
    "2f02d20a-8dca-49b7-b26f-b6ce9fddaf4f",
    "ca1e5ed7-dae8-4605-987b-fadaeeb30461",
  ]
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"

  network {
    port = "${openstack_networking_port_v2.port_1.id}"
  }

  connection {
    user        = "root"
    host        = "${openstack_networking_port_v2.port_1.fixed_ip.0.ip_address}"
    private_key = "~/path/to/key"
  }

  provisioner "remote-exec" {
    inline = [
      "echo terraform executed > /tmp/foo",
    ]
  }
}
```
