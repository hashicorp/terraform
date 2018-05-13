---
layout: "functions"
page_title: "cidrhost function"
sidebar_current: "docs-funcs-ipnet-cidrhost"
description: |-
  The cidrhost function calculates a full host IP address within a given
  IP network address prefix.
---

# `cidrhost` Function

`cidrhost` calculates a full host IP address for a given host number within
a given IP network address prefix.

```hcl
cidrhost(prefix, hostnum)
```

`prefix` must be given in CIDR notation, as defined in
[RFC 4632 section 3.1](https://tools.ietf.org/html/rfc4632#section-3.1).

`hostnum` is a whole number that can be represented as a binary integer with
no more than the number of digits remaining in the address after the given
prefix.

This function accepts both IPv6 and IPv4 prefixes, and the result always uses
the same addressing scheme as the given prefix.

## Examples

```
> cidrhost("10.12.127.0/20", 16)
10.12.112.16
> cidrhost("10.12.127.0/20", 268)
10.12.113.12
> cidrhost("fd00:fd12:3456:7890:00a2::/72", 34)
fd00:fd12:3456:7890::22
```

## Related Functions

* [`cidrsubnet`](./cidrsubnet.html) calculates a subnet address under a given
  network address prefix.
