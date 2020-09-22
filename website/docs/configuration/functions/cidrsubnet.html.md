---
layout: "functions"
page_title: "cidrsubnet - Functions - Configuration Language"
sidebar_current: "docs-funcs-ipnet-cidrsubnet"
description: |-
  The cidrsubnet function calculates a subnet address within a given IP network
  address prefix.
---

# `cidrsubnet` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`cidrsubnet` calculates a subnet address within given IP network address prefix.

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

Unlike the related function [`cidrsubnets`](./cidrsubnets.html), `cidrsubnet`
allows you to give a specific network number to use. `cidrsubnets` can allocate
multiple network addresses at once, but numbers them automatically starting
with zero.

## Examples

```
> cidrsubnet("172.16.0.0/12", 4, 2)
172.18.0.0/16
> cidrsubnet("10.1.2.0/24", 4, 15)
10.1.2.240/28
> cidrsubnet("fd00:fd12:3456:7890::/56", 16, 162)
fd00:fd12:3456:7800:a200::/72
```

## Netmasks and Subnets

Using `cidrsubnet` requires familiarity with some network addressing concepts.

The most important idea is that an IP address (whether IPv4 or IPv6) is
fundamentally constructed from binary digits, even though we conventionally
represent it as either four decimal octets (for IPv4) or a sequence of 16-bit
hexadecimal numbers (for IPv6).

Taking our example above of `cidrsubnet("10.1.2.0/24", 4, 15)`, the function
will first convert the given IP address string into an equivalent binary
representation:

```
      10 .        1 .        2 .        0
00001010   00000001   00000010 | 00000000
         network               |   host
```

The `/24` at the end of the prefix string specifies that the first 24
bits -- or, the first three octets -- of the address identify the network
while the remaining bits (32 - 24 = 8 bits in this case) identify hosts
within the network.

The CLI tool [`ipcalc`](https://gitlab.com/ipcalc/ipcalc) is useful for
visualizing CIDR prefixes as binary numbers. We can confirm the conversion
above by providing the same prefix string to `ipcalc`:

```
$ ipcalc 10.1.2.0/24
Address:   10.1.2.0             00001010.00000001.00000010. 00000000
Netmask:   255.255.255.0 = 24   11111111.11111111.11111111. 00000000
Wildcard:  0.0.0.255            00000000.00000000.00000000. 11111111
=>
Network:   10.1.2.0/24          00001010.00000001.00000010. 00000000
HostMin:   10.1.2.1             00001010.00000001.00000010. 00000001
HostMax:   10.1.2.254           00001010.00000001.00000010. 11111110
Broadcast: 10.1.2.255           00001010.00000001.00000010. 11111111
Hosts/Net: 254                   Class A, Private Internet
```

This gives us some additional information but also confirms (using a slightly
different notation) the conversion from decimal to binary and shows the range
of possible host addresses in this network.

While [`cidrhost`](./cidrhost.html) allows calculating single host IP addresses,
`cidrsubnet` on the other hand creates a new network prefix _within_ the given
network prefix. In other words, it creates a subnet.

When we call `cidrsubnet` we also pass two additional arguments: `newbits` and
`netnum`. `newbits` decides how much longer the resulting prefix will be in
bits; in our example here we specified `4`, which means that the resulting
subnet will have a prefix length of 24 + 4 = 28 bits. We can imagine these
bits breaking down as follows:

```
      10 .        1 .        2 .    ?        0
00001010   00000001   00000010 |   XXXX | 0000
         parent network        | netnum | host
```

Four of the eight bits that were originally the "host number" are now being
repurposed as the subnet number. The network prefix no longer falls on an
exact octet boundary, so in effect we are now splitting the last decimal number
in the IP address into two parts, using half of it to represent the subnet
number and the other half to represent the host number.

The `netnum` argument then decides what number value to encode into those
four new subnet bits. In our current example we passed `15`, which is
represented in binary as `1111`, allowing us to fill in the `XXXX` segment
in the above:

```
      10 .        1 .        2 .    15       0
00001010   00000001   00000010 |   1111 | 0000
         parent network        | netnum | host
```

To convert this back into normal decimal notation we need to recombine the
two portions of the final octet. Converting `11110000` from binary to decimal
gives 240, which can then be combined with our new prefix length of 28 to
produce the result `10.1.2.240/28`. Again we can pass this prefix string to
`ipcalc` to visualize it:

```
$ ipcalc 10.1.2.240/28
Address:   10.1.2.240           00001010.00000001.00000010.1111 0000
Netmask:   255.255.255.240 = 28 11111111.11111111.11111111.1111 0000
Wildcard:  0.0.0.15             00000000.00000000.00000000.0000 1111
=>
Network:   10.1.2.240/28        00001010.00000001.00000010.1111 0000
HostMin:   10.1.2.241           00001010.00000001.00000010.1111 0001
HostMax:   10.1.2.254           00001010.00000001.00000010.1111 1110
Broadcast: 10.1.2.255           00001010.00000001.00000010.1111 1111
Hosts/Net: 14                    Class A, Private Internet
```

The new subnet has four bits available for host numbering, which means
that there are 14 host addresses available for assignment once we subtract
the network's own address and the broadcast address. You can thus use
[`cidrhost`](./cidrhost.html) function to calculate those host addresses by
providing it a value between 1 and 14:

```
> cidrhost("10.1.2.240/28", 1)
10.1.2.241
> cidrhost("10.1.2.240/28", 14)
10.1.2.254
```

For more information on CIDR notation and subnetting, see
[Classless Inter-domain Routing](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing).

## Related Functions

* [`cidrhost`](./cidrhost.html) calculates the IP address for a single host
  within a given network address prefix.
* [`cidrnetmask`](./cidrnetmask.html) converts an IPv4 network prefix in CIDR
  notation into netmask notation.
* [`cidrsubnets`](./cidrsubnets.html) can allocate multiple consecutive
  addresses under a prefix at once, numbering them automatically.
