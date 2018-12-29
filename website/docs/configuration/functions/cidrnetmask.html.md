---
layout: "functions"
page_title: "cidrnetmask - Functions - Configuration Language"
sidebar_current: "docs-funcs-ipnet-cidrnetmask"
description: |-
  The cidrnetmask function converts an IPv4 address prefix given in CIDR
  notation into a subnet mask address.
---

# `cidrnetmask` Function

`cidrnetmask` converts an IPv4 address prefix given in CIDR notation into
a subnet mask address.

```hcl
cidrnetmask(prefix)
```

`prefix` must be given in IPv4 CIDR notation, as defined in
[RFC 4632 section 3.1](https://tools.ietf.org/html/rfc4632#section-3.1).

The result is a subnet address formatted in the conventional dotted-decimal
IPv4 address syntax, as expected by some software.

CIDR notation is the only valid notation for IPv6 addresses, so `cidrnetmask`
produces an error if given an IPv6 address.

## Examples

```
> cidrnetmask("172.16.0.0/12")
255.240.0.0
```
