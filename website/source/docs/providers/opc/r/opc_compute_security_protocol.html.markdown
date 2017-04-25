---
layout: "opc"
page_title: "Oracle: opc_compute_security_protocol"
sidebar_current: "docs-opc-resource-security-protocol"
description: |-
  Creates and manages an security protocol in an OPC identity domain.
---

# opc\_compute\_security\_protocol

The ``opc_compute_security_protocol`` resource creates and manages a security protocol in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_security_protocol" "default" {
  name        = "security-protocol-1"
  dst_ports   = ["2045-2050"]
  src_ports   = ["3045-3060"]
  ip_protocol = "tcp"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the security protocol.

* `dst_ports` (Optional) Enter a list of port numbers or port range strings.
	Traffic is enabled by a security rule when a packet's destination port matches the
  ports specified here.
	For TCP, SCTP, and UDP, each port is a destination transport port, between 0 and 65535,
	inclusive. For ICMP, each port is an ICMP type, between 0 and 255, inclusive.
	If no destination ports are specified, all destination ports or ICMP types are allowed.

* `src_ports` (Optional) Enter a list of port numbers or port range strings.
	Traffic is enabled by a security rule when a packet's source port matches the
	ports specified here.
	For TCP, SCTP, and UDP, each port is a source transport port,
	between 0 and 65535, inclusive.
	For ICMP, each port is an ICMP type, between 0 and 255, inclusive.
	If no source ports are specified, all source ports or ICMP types are allowed.

* `ip_protocol` (Optional) The protocol used in the data portion of the IP datagram.
	 Permitted values are: tcp, udp, icmp, igmp, ipip, rdp, esp, ah, gre, icmpv6, ospf, pim, sctp,
	 mplsip, all.
	 Traffic is enabled by a security rule when the protocol in the packet matches the
	 protocol specified here. If no protocol is specified, all protocols are allowed.

* `description` - (Optional) A description of the security protocol.

* `tags` - (Optional) List of tags that may be applied to the security protocol.

In addition to the above, the following values are exported:

* `uri` - The Uniform Resource Identifier for the Security Protocol

## Import

ACL's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_security_protocol.default example
```
