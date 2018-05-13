---
layout: "functions"
page_title: "cidrsubnet function"
sidebar_current: "docs-funcs-ipnet-cidrsubnet"
description: |-
  The cidrsubnet function calculates a subnet address within a given IP network
  address prefix.
---

# `cidrsubnet` Function

`cidrhost` calculates a subnet address within given IP network address prefix.

```hcl
cidrsubnet(prefix, newbits, netnum)
```

`prefix` must be given in CIDR notation, as defined in
[RFC 4632 section 3.1](https://tools.ietf.org/html/rfc4632#section-3.1).

`newbits` is the number of additional bits with which to extend the prefix.
For example, if given a prefix ending in `/16` and a `newbits` value of
`4`, the resulting subnet address will have length `/20`.

`netnum` is a whole number that can be represented as a binary integer with
no more than `newbits` binary digits, which will be used to populate the
additional bits added to the prefix.

This function accepts both IPv6 and IPv4 prefixes, and the result always uses
the same addressing scheme as the given prefix.

## Examples

```
> cidrsubnet("172.16.0.0/12", 4, 2)
172.18.0.0/16
> cidrsubnet("10.1.2.0/24", 4, 15)
10.1.2.240/28
> cidrsubnet("fd00:fd12:3456:7890::/56", 16, 162)
fd00:fd12:3456:7800:a200::/72
```

## Related Functions

* [`cidrhost`](./cidrhost.html) calculates the IP address for a single host
  within a given network address prefix.
* [`cidrnetmask`](./cidrnetmask.html) converts an IPv4 network prefix in CIDR
  notation into netmask notation.
