---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_ssh_keypair"
sidebar_current: "docs-cloudstack-resource-ssh-keypair"
description: |-
  Creates or registers an SSH keypair.
---

# cloudstack\_ssh\_keypair

Creates or registers an SSH keypair.

## Example Usage

```
resource "cloudstack_ssh_keypair" "myKey" {
  name = "myKey"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name to give the SSH keypair. This is a unique value within a Cloudstack account.

* `public_key` - (Optional) The full public key text of this keypair. If this is omitted, Cloudstack 
  will generate a new keypair.

## Attributes Reference

The following attributes are exported:

* `id` - The keypair ID. This is set to the keypair `name` argument.
* `fingerprint` - The fingerprint of the public key specified or calculated.
* `private_key` - This is returned only if Cloudstack generated the keypair.
