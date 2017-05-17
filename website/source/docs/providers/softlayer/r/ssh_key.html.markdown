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

```hcl
resource "softlayer_ssh_key" "test_ssh_key" {
  name       = "test_ssh_key_name"
  notes      = "test_ssh_key_notes"
  public_key = "ssh-rsa <rsa_public_key>"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A descriptive name used to identify an SSH key.
* `public_key` - (Required) The public SSH key.
* `notes` - (Optional) A small note about an SSH key to use at your discretion.

The `name` and `notes` fields are editable.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the new SSH key
* `fingerprint` - sequence of bytes to authenticate or lookup a longer SSH key.
