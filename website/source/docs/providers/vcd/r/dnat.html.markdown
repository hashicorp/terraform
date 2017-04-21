---
layout: "vcd"
page_title: "vCloudDirector: vcd_dnat"
sidebar_current: "docs-vcd-resource-dnat"
description: |-
  Provides a vCloud Director DNAT resource. This can be used to create, modify, and delete destination NATs to map external IPs to a VM.
---

# vcd\_dnat

Provides a vCloud Director DNAT resource. This can be used to create, modify,
and delete destination NATs to map an external IP/port to a VM.

## Example Usage

```hcl
resource "vcd_dnat" "web" {
  edge_gateway = "Edge Gateway Name"
  external_ip  = "78.101.10.20"
  port         = 80
  internal_ip  = "10.10.0.5"
}
```

## Argument Reference

The following arguments are supported:

* `edge_gateway` - (Required) The name of the edge gateway on which to apply the DNAT
* `external_ip` - (Required) One of the external IPs available on your Edge Gateway
* `port` - (Required) The port number to map
* `internal_ip` - (Required) The IP of the VM to map to
