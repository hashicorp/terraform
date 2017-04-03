---
layout: "oracleopc"
page_title: "Oracle: opc_compute_security_application"
sidebar_current: "docs-oracleopc-resource-security-application"
description: |-
  Creates and manages a security application in an OPC identity domain.
---

# opc\_compute\_security\_application

The ``opc_compute_security_application`` resource creates and manages a security application in an OPC identity domain.

## Example Usage

```
resource "opc_compute_security_application" "tomcat" {
	name = "tomcat"
	protocol = "tcp"
	dport = "8080"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within the identity domain) name of the application

* `protocol` - (Required) The protocol to enable for this application. Must be either one of
`tcp`, `udp`, `icmp`, `igmp`, `ipip`, `rdp`, `esp`, `ah`, `gre`, `icmpv6`, `ospf`, `pim`, `sctp`, `mplsip` or `all`, or
the corresponding integer in the range 0-254 from the list of [assigned protocol numbers](http://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml)

* `dport` - (Required) The port, or range of ports, to enable for this application, e.g `8080`, `6000-7000`.

* `icmptype` - (Optional) The ICMP type to enable for this application, if the `protocol` is `icmp`. Must be one of
`echo`, `reply`, `ttl`, `traceroute`, `unreachable`.

* `icmpcode` - (Optional) The ICMP code to enable for this application, if the `protocol` is `icmp`. Must be one of
`network`, `host`, `protocol`, `port`, `df`, `admin`.
