---
layout: "language"
page_title: "cidrnet - Functions - Configuration Language"
sidebar_current: "docs-funcs-ipnet-cidrnet"
description: |-
  The cidrnet function converts an IPv4 address
  and subnet mask in conventional dotted-decimal
  notation into a CIDR prefix
---

# `cidrnet` Function

`cidrnet` converts an IPv4 address and subnet mask in conventional dotted-decimal 
IPv4 address syntax into it's CIDR prefix

```hcl
cidrnet(address, subnetmask)
```

`address` must be given in conventional dotted-decimal IPv4 address syntax
```
"192.168.1.0"
```

`subnetmask` must be given in conventional dotted-decimal IPv4 address syntax
```
"255.255.255.255"
```

The result is the CIDR prefix for the given subnet

CIDR notation is the only valid notation for IPv6 addresses, so `cidrnet`
has no support for IPv6

## Examples

```
> cidrnet("192.168.1.6", "255.255.254.0")
192.168.0.0/23
> cidrnet("192.168.0.200", "255.255.255.128")
192.168.0.128/25
> cidrnet("192.168.1.6", "255.255.255.255")
192.168.1.6/32
```


## Related Functions

* [`cidrbitmask`](./cidrbitmask.html) calculates the prefix size
given a subnet mask in conventional dotted-decimal IPv4 address syntax
