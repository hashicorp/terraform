---
layout: "openstack"
page_title: "OpenStack: openstack_compute_keypair"
sidebar_current: "docs-openstack-resource-compute-keypair"
description: |-
  Manages a keypair resource within OpenStack.
---

# openstack\_compute\_keypair

Manages a keypair resource within OpenStack.

## Example Usage

```
resource "openstack_compute_keypair" "test-keypair" {
  name = "my-keypair"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDAjpC1hwiOCCmKEWxJ4qzTTsJbKzndLotBCz5PcwtUnflmU+gHJtWMZKpuEGVi29h0A/+ydKek1O18k10Ff+4tyFjiHDQAnOfgWf7+b1yK+qDip3X1C0UPMbwHlTfSGWLGZqd9LvEFx9k3h/M+VtMvwR1lJ9LUyTAImnNjWG7TaIPmui30HvM2UiFEmqkr4ijq45MyX2+fLIePLRIF61p4whjHAQYufqyno3BS48icQb4p6iVEZPo4AE2o9oIyQvj2mx4dk5Y8CgSETOZTYDOR3rU2fZTRDRgPJDH9FWvQjF5tA0p3d9CoWWd2s6GKKbfoUIi8R/Db1BSPJwkqB"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the keypair. Changing this creates a new
    keypair.

* `public_key` - (Required) A pregenerated OpenSSH-formatted public key.
    Changing this creates a new keypair.

## Attributes Reference

The following attributes are exported:

* `name` - See Argument Reference above.
* `public_key` - See Argument Reference above.
