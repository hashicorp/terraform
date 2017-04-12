---
layout: "rundeck"
page_title: "Rundeck: rundeck_private_key"
sidebar_current: "docs-rundeck-resource-private-key"
description: |-
  The rundeck_private_key resource allows private keys to be stored in Rundeck's key store.
---

# rundeck\_private\_key

The private key resource allows SSH private keys to be stored into Rundeck's key store.
The key store is where Rundeck keeps credentials that are needed to access the nodes on which
it runs commands.

## Example Usage

```hcl
resource "rundeck_private_key" "anvils" {
    path = "anvils/id_rsa"
    key_material = "${file("/id_rsa")}"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path within the key store where the key will be stored.

* `key_material` - (Required) The private key material to store, serialized in any way that is
  accepted by OpenSSH.

The key material is hashed before it is stored in the state file, so sharing the resulting state
will not disclose the private key contents.

## Attributes Reference

Rundeck does not allow stored private keys to be retrieved via the API, so this resource does not
export any attributes.
