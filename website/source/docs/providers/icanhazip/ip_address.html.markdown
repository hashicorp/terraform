---
layout: "icanhazip"
page_title: "icanhazip: icanhazip_ip_address"
sidebar_current: "docs-icanhazip-datasource-ip_address"
description: |-
  Get your external IP IPv4 or IPv6 Address.
---

# icanhazip\_ip_address

Use this data source to get the IP address you appear as on the internet.

## Example Usage

```hcl
data "icanhazip_ipaddress" "localip" { }

resource "aws_security_group" "from_office" {
  name = "from_office"

  ingress {
    from_port   = "22"
    to_port     = "22"
    protocol    = "tcp"
    cidr_blocks = ["${data.icanhazip_ipaddress.localip.ip_address}/32"]
  }
}
```

## Argument Reference

* `version` - (Optional) The version of the IP protocol to fetch your IP address
  for. Valid versions are `ipv4`, the default, and `ipv6`.

## Attributes Reference

* `ip_address` - The IP address you present to the internet.

