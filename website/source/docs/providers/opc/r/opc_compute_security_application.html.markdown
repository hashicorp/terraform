---
layout: "opc"
page_title: "Oracle: opc_compute_security_application"
sidebar_current: "docs-opc-resource-security-application"
description: |-
  Creates and manages a security application in an OPC identity domain.
---

# opc\_compute\_security\_application

The ``opc_compute_security_application`` resource creates and manages a security application in an OPC identity domain.

## Example Usage (TCP)

```hcl
resource "opc_compute_security_application" "tomcat" {
  name     = "tomcat"
  protocol = "tcp"
  dport    = "8080"
}
```

## Example Usage (ICMP)

```hcl
resource "opc_compute_security_application" "tomcat" {
  name     = "tomcat"
  protocol = "icmp"
  icmptype = "echo"
  icmpcode = "protocol"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within the identity domain) name of the application

* `protocol` - (Required) The protocol to enable for this application. Must be one of
`tcp`, `udp`, `ah`, `esp`, `icmp`, `icmpv6`, `igmp`, `ipip`, `gre`, `mplsip`, `ospf`, `pim`, `rdp`, `sctp` or `all`.

* `dport` - (Required) The port, or range of ports, to enable for this application, e.g `8080`, `6000-7000`. This must be set if the `protocol` is set to `tcp` or `udp`.

* `icmptype` - (Optional) The ICMP type to enable for this application, if the `protocol` is `icmp`. Must be one of
`echo`, `reply`, `ttl`, `traceroute`, `unreachable`.

* `icmpcode` - (Optional) The ICMP code to enable for this application, if the `protocol` is `icmp`. Must be one of
`admin`, `df`, `host`, `network`, `port` or `protocol`.

## Import

Security Application's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_security_application.application1 example
```
