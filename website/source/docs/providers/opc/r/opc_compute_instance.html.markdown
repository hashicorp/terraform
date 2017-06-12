---
layout: "opc"
page_title: "Oracle: opc_compute_instance"
sidebar_current: "docs-opc-resource-instance"
description: |-
  Creates and manages an instance in an OPC identity domain.
---

# opc\_compute\_instance

The ``opc_compute_instance`` resource creates and manages an instance in an OPC identity domain.

~> **Caution:** The ``opc_compute_instance`` resource can completely delete your
instance just as easily as it can create it. To avoid costly accidents,
consider setting
[``prevent_destroy``](/docs/configuration/resources.html#prevent_destroy)
on your instance resources as an extra safety measure.

## Example Usage

```hcl
resource "opc_compute_ip_network" "test" {
  name                = "internal-network"
  description         = "Terraform Provisioned Internal Network"
  ip_address_prefix   = "10.0.1.0/24"
  public_napt_enabled = false
}

resource "opc_compute_storage_volume" "test" {
  name = "internal"
  size = 100
}

resource "opc_compute_instance" "test" {
  name       = "instance1"
  label      = "Terraform Provisioned Instance"
  shape      = "oc3"
  image_list = "/oracle/public/oel_6.7_apaas_16.4.5_1610211300"

  storage {
    volume = "${opc_compute_storage_volume.test.name}"
    index  = 1
  }

  networking_info {
    index          = 0
    nat            = ["ippool:/oracle/public/ippool"]
    shared_network = true
  }
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the instance.

* `shape` - (Required) The shape of the instance, e.g. `oc4`.

* `instance_attributes` - (Optional) A JSON string of custom attributes. See [Attributes](#attributes) below for more information.

* `boot_order` - (Optional) The index number of the bootable storage volume, presented as a list, that should be used to boot the instance. The only valid value is `[1]`. If you set this attribute, you must also specify a bootable storage volume with index number 1 in the volume sub-parameter of storage_attachments. When you specify boot_order, you don't need to specify the imagelist attribute, because the instance is booted using the image on the specified bootable storage volume. If you specify both boot_order and imagelist, the imagelist attribute is ignored.

* `hostname` - (Optional) The host name assigned to the instance. On an Oracle Linux instance, this host name is displayed in response to the hostname command. Only relative DNS is supported. The domain name is suffixed to the host name that you specify. The host name must not end with a period. If you don't specify a host name, then a name is generated automatically.

* `image_list` - (Optional) The imageList of the instance, e.g. `/oracle/public/oel_6.4_2GB_v1`.

* `label` - (Optional) The label to apply to the instance.

* `networking_info` - (Optional) Information pertaining to an individual network interface to be created and attached to the instance. See [Networking Info](#networking-info) below for more information.

* `storage` - (Optional) Information pertaining to an individual storage attachment to be created during instance creation. Please see [Storage Attachments](#storage-attachments) below for more information.

* `reverse_dns` - (Optional) If set to `true` (default), then reverse DNS records are created. If set to `false`, no reverse DNS records are created.

* `ssh_keys` - (Optional) A list of the names of the SSH Keys that can be used to log into the instance.  

* `tags` - (Optional) A list of strings that should be supplied to the instance as tags.

## Attributes

During instance creation, there are several custom attributes that a user may wish to make available to the instance during instance creation.
These attributes can be specified via the `instance_attributes` field, and must be presented as a string in JSON format.
The easiest way to populate this field is with a HEREDOC:

```hcl
resource "opc_compute_instance" "foo" {
  name = "test"
  label = "test"
  shape = "oc3"
  imageList = "/oracle/public/oel_6.4_2GB_v1"
  instance_attributes = <<JSON
{
  "foo": "bar",
  "baz": 42,
  "my_obj": {
    "my_key": false,
    "another": true
  }
}
JSON

  sshKeys = ["${opc_compute_ssh_key.key1.name}"]
}
```

This allows the user to have full control over the attributes supplied to an instance during instance creation.
There are, as well, some attributes that get populated during instance creation, and the full attributes map can be seen
via the exported `attributes` attribute.

**Warning:** Due to how Terraform imports resources, the `instance_attributes` field will _only_ be populated
when creating a new instance _with terraform_. This requires us to ignore any state diffs on changes to the `instance_attributes` field.
Thus, any configuration changes in the `instance_attributes` field, will not register a diff during a `plan` or `apply`.
 If a user wishes to make a change solely to the supplied instance attributes, and recreate the instance resource, `terraform taint` is the best solution.
 You can read more about the `taint` command [here](https://www.terraform.io/docs/commands/taint.html)

## Networking Info

Each `networking_info` config manages a single network interface for the instance.
The attributes are either required or optional depending on whether or not the interface is
in the Shared Network, or an IP Network. Some attributes can only be used if the interface is in the Shared
 Network, and same for an interface in an IP Network.

The following attributes are supported:

* `index` - (Required) The numerical index of the network interface. Specified as an integer to allow for use of `count`, but directly maps to `ethX`. ie: With `index` set to `0`, the interface `eth0` will be created. Can only be `0-9`.
* `dns` - (Optional) Array of DNS servers for the interface.
* `ip_address` - (Optional, IP Network Only) IP Address assigned to the interface.
* `ip_network` - (Optional, IP Network Only) The IP Network assigned to the interface.
* `mac_address` - (Optional, IP Network Only) The MAC address of the interface.
* `model` - (Required, Shared Network Only) The model of the NIC card used. Must be set to `e1000`.
* `name_servers` - (Optional) Array of name servers for the interface.
* `nat` - (Optional for IP Networks, Required for the Shared Network) The IP Reservations associated with the interface (IP Network).
 Indicates whether a temporary or permanent public IP address should be assigned to the instance (Shared Network).
* `search_domains` - (Optional) The search domains that are sent through DHCP as option 119.
* `sec_lists` - (Optional, Shared Network Only) The security lists the interface is added to.
* `shared_network` - (Required) Whether or not the interface is inside the Shared Network or an IP Network.
* `vnic` - (Optional, IP Network Only) The name of the vNIC created for the IP Network.
* `vnic_sets` - (Optional, IP Network Only) The array of vNIC Sets the interface was added to.

## Storage Attachments

Each Storage Attachment config manages a single storage attachment that is created _during instance creation_.
This means that any storage attachments created during instance creation cannot be detached from the instance.
Use the `resource_storage_attachment` resource to manage storage attachments for instances if you wish to detach the
storage volumes at a later date.

The following attributes are supported:

* `index` - (Required) The Index number of the volume attachment. `1` is the boot volume for the instance. Values `1-10` allowed.
* `volume` - (Required) The name of the storage volume to attach to the instance.

In addition to the above attributes, the following attributes are exported for a storage volume

* `name` - Name of the storage volume attachment.

## Attributes Reference

In addition to the attributes listed above, the following attributes are exported:

* `id` - The `id` of the instance.
* `attributes` - The full attributes of the instance, as a JSON string.
* `availability_domain` - The availability domain the instance is in.
* `domain` - The default domain to use for the hostname and for DNS lookups.
* `entry` - Imagelist entry number.
* `fingerprint` - SSH server fingerprint presented by the instance.
* `fqdn` - The fully qualified domain name of the instance.
* `image_format` - The format of the image.
* `ip_address` - The IP Address of the instance.
* `placement_requirements` - The array of placement requirements for the instance.
* `platform` - The OS Platform of the instance.
* `priority` - The priority at which the instance was ran.
* `quota_reservation` - Reference to the QuotaReservation, to be destroyed with the instance.
* `relationships` - The array of relationship specifications to be satisfied on instance placement.
* `resolvers` - Array of resolvers to be used instead of the default resolvers.
* `site` - The site the instance is running on.
* `start_time` - The launch time of the instance.
* `state` - The instance's state.
* `vcable_id` - vCable ID for the instance.
* `virtio` - Boolean that determines if the instance is a virtio device.
* `vnc_address` - The VNC address and port of the instance.

## Import

Instances can be imported using the Instance's combined `Name` and `ID` with a `/` character separating them.
If viewing an instance in the Oracle Web Console, the instance's `name` and `id` are the last two fields in the instances fully qualified `Name`

For example, in the Web Console an instance's fully qualified name is:
```
/Compute-<identify>/<user>@<account>/<instance_name>/<instance_id>
```

The instance can be imported as such:

```shell
$ terraform import opc_compute_instance.instance1 instance_name/instance_id
```
