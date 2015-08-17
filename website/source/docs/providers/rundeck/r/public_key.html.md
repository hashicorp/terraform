---
layout: "rundeck"
page_title: "Rundeck: rundeck_public_key"
sidebar_current: "docs-rundeck-resource-public-key"
description: |-
  The rundeck_public_key resource allows public keys to be stored in Rundeck's key store.
---

# rundeck\_public\_key

The public key resource allows SSH public keys to be stored into Rundeck's key store.
The key store is where Rundeck keeps credentials that are needed to access the nodes on which
it runs commands.

This resource also allows the retrieval of an existing public key from the store, so that it
may be used in the configuration of other resources such as ``aws_key_pair``.

## Example Usage

```
resource "rundeck_public_key" "anvils" {
    path = "anvils/id_rsa.pub"
    key_material = "ssh-rsa yada-yada-yada"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path within the key store where the key will be stored. By convention
  this path name normally ends with ".pub" and otherwise has the same name as the associated
  private key.

* `key_material` - (Optional) The public key string to store, serialized in any way that is accepted
  by OpenSSH. If this is not included, ``key_material`` becomes an attribute that can be used
  to read the already-existing key material in the Rundeck store.

The key material is included inline as a string, which is consistent with the way a public key
is provided to the `aws_key_pair`, `cloudstack_ssh_keypair`, `digitalocean_ssh_key` and
`openstack_compute_keypair_v2` resources. This means the `key_material` argument can be populated
from the interpolation of the `public_key` attribute of such a keypair resource, or vice-versa.

## Attributes Reference

The following attributes are exported:

* `url` - The URL at which the key material can be retrieved from the key store by other clients.

* `key_material` - If `key_material` is omitted in the configuration, it becomes an attribute that
  exposes the key material already stored at the given `path`.
