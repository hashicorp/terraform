---
layout: "packet"
page_title: "Packet: packet_ssh_key"
sidebar_current: "docs-packet-resource-ssh-key"
description: |-
  Provides a Packet SSH key resource.
---

# packet\_ssh_key

Provides a Packet SSH key resource to allow you manage SSH
keys on your account. All ssh keys on your account are loaded on
all new devices, they do not have to be explicitly declared on
device creation.

## Example Usage

```
# Create a new SSH key
resource "packet_ssh_key" "key1" {
    name = "terraform-1"
    public_key = "${file("/home/terraform/.ssh/id_rsa.pub")}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the SSH key for identification
* `public_key` - (Required) The public key. If this is a file, it
can be read using the file interpolation function

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the key
* `name` - The name of the SSH key
* `public_key` - The text of the public key
* `fingerprint` - The fingerprint of the SSH key
* `created` - The timestamp for when the SSH key was created
* `updated` - The timestamp for the last time the SSH key was udpated
