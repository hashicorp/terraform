---
layout: "clc"
page_title: "clc: clc_public_ip"
sidebar_current: "docs-clc-resource-public-ip"
description: |-
  Manages a CLC public ip.
---

# clc\_public\_ip

Manages a CLC public ip (for an existing server).

See also [Complete API documentation](https://www.ctl.io/api-docs/v2/#public-ip).

## Example Usage

```
# Provision a public ip
resource "clc_public_ip" "backdoor" {
  server_id = "${clc_server.node.0.id}"
  internal_ip_address = "${clc_server.node.0.private_ip_address}"
  ports
    {
      protocol = "ICMP"
      port = -1
    }
  ports
    {
      protocol = "TCP"
      port = 22
    }
  ports
    {
      protocol = "TCP"
      port = 2000
      port_to = 9000
    }
  source_restrictions
     { cidr = "85.39.22.15/30" }
}


output "ip" {
  value = "clc_public_ip.backdoor.id"
}

```

## Argument Reference

The following arguments are supported:

* `server_id` - (Required, string) The name or ID of the server to bind IP to.
* `internal_ip_address` - (Required, string) The internal IP of the
  NIC to attach to. If not provided, a new internal NIC will be
  provisioned and used.
* `ports` - (Optional) See [Ports](#ports) below for details.
* `source_restrictions` - (Optional) See
  [SourceRestrictions](#source_restrictions) below for details.


<a id="ports"></a>
## Ports

`ports` is a block within the configuration that may be
repeated to specify open ports on the target IP. Each
`ports` block supports the following:

* `protocol` (Required, string) One of "tcp", "udp", "icmp".
* `port` (Required, int) The port to open. If defining a range, demarks starting port
* `portTo` (Optional, int) Given a port range, demarks the ending port. 


<a id="source_restrictions"></a>
## SourceRestrictions

`source_restrictions` is a block within the configuration that may be
repeated to restrict ingress traffic on specified CIDR blocks. Each
`source_restrictions` block supports the following:

* `cidr` (Required, string) The IP or range of IPs in CIDR notation.




