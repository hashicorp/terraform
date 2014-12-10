---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_port_forward"
sidebar_current: "docs-cloudstack-resource-port-forward"
description: |-
  Creates port forwards.
---

# cloudstack\_port\_forward

Creates port forwards.

## Example Usage

```
resource "cloudstack_port_forward" "default" {
  ipaddress = "192.168.0.1"

  forward {
    protocol = "tcp"
    private_port = 80
    public_port = 8080
    virtual_machine = "server-1"
  }
}
```

## Argument Reference

The following arguments are supported:

* `ipaddress` - (Required) The ip address for which to create the port forwards.
    Changing this forces a new resource to be created.

* `forward` - (Required) Can be specified multiple times. Each forward block supports
    fields documented below.

The `forward` block supports:

* `protocol` - (Required) The name of the protocol to allow. Valid options are:
    `tcp` and `udp`.

* `private_port` - (Required) The private port to forward to.

* `public_port` - (Required) The public port to forward from.

* `virtual_machine` - (Required) The name of the virtual machine to forward to.

## Attributes Reference

The following attributes are exported:

* `ipaddress` - The ip address for which the port forwards are created.
