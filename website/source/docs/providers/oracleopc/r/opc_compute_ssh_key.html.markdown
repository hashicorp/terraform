---
layout: "oracleopc"
page_title: "Oracle: opc_compute_ssh_key"
sidebar_current: "docs-oracleopc-resource-ssh-key"
description: |-
  Creates and manages an SSH key in an OPC identity domain.
---

# opc\_compute\_ssh_key

The ``opc_compute_ssh_key`` resource creates and manages an SSH key in an OPC identity domain.

## Example Usage

```
resource "opc_compute_ssh_key" "%s" {
	name = "test-key"
	key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCqw6JwbjIk..."
	enabled = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within this identity domain) name of the SSH key.

* `key` - (Required) The SSH key itself

* `enabled` - (Required) Whether or not the key is enabled. This is useful if you want to temporarily disable an SSH key,
without removing it entirely from your Terraform resource definition.
