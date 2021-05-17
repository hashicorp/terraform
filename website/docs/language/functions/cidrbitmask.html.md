---
layout: "language"
page_title: "cidrbitmask - Functions - Configuration Language"
sidebar_current: "docs-funcs-ipnet-cidrbitmask"
description: |-
  The cidrbitmask function converts an IPv4 subnet mask in decimal dot
  notation into a CIDR mask size
---

# `cidrbitmask` Function

`cidrbitmask` converts an IPv4 subnet mask in conventional dotted-decimal 
IPv4 address syntax into a CIDR mask size

```hcl
cidrbitmask(subnetmask)
```

`subnetmask` must be given in conventional dotted-decimal IPv4 address syntax
```
"255.255.255.255"
```

The result is a CIDR mask size as an integer

CIDR notation is the only valid notation for IPv6 addresses, so `cidrbitmask`
produces an error if given an IPv6 address.

## Examples

```
> cidrbitmask("255.240.0.0")
12
> cidrbitmask("255.255.255.0")
24
> cidrbitmask("255.255.255.255")
32
```

## Related Functions

* [`cidrnet`](./cidrnet.html) calculates the CIDR prefix given an address
and subnet mask in conventional dotted-decimal IPv4 address syntax