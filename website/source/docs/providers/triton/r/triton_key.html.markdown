---
layout: "triton"
page_title: "Triton: triton_key"
sidebar_current: "docs-triton-firewall"
description: |-
    The `triton_key` resource represents an SSH key for a Triton account. 
---

# triton\_key

The `triton_key` resource represents an SSH key for a Triton account.

## Example Usages

Create a key


```
resource "triton_key" "example" {
    name = "Example Key"
    key = "${file("keys/id_rsa")}"
}
                
```

## Argument Reference

The following arguments are supported:

* `name` - (string, Change forces new resource)
    The name of the key. If this is left empty, the name is inferred from the comment in the SSH key material.

* `key` - (string, Required, Change forces new resource)
    The SSH key material. In order to read this from a file, use the `file` interpolation.

