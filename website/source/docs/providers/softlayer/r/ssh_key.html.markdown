---
layout: "softlayer"
page_title: "SoftLayer: ssh_key"
sidebar_current: "docs-softlayer-resource-ssh-key"
description: |-
  Manages SoftLayer SSH Keys.
---

# softlayer\ssh_key

Provides SSK keys. This allows SSH keys to be created, updated and deleted.
For additional details please refer to [API documentation](http://sldn.softlayer.com/reference/datatypes/SoftLayer_Security_Ssh_Key).

## Example Usage

```
resource "softlayer_ssh_key" "test_ssh_key" {
    name = "test_ssh_key_name"
    notes = "test_ssh_key_notes"
    public_key = "ssh-rsa <rsa_public_key>"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A descriptive name used to identify a ssh key.
* `public_key` - (Required) The public ssh key.
* `notes` - (Optional) A small note about a ssh key to use at your discretion.

Fields `name` and `notes` are editable.

## Attributes Reference

The following attributes are exported:

* `id` - id of the new ssh key
* `fingerprint` - sequence of bytes to authenticate or lookup a longer ssh key.
