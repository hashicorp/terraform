---
layout: "fastly"
page_title: "Fastly: fastly_ip_ranges"
sidebar_current: "docs-fastly-datasource-ip_ranges"
description: |-
  Get information on Fastly IP ranges.
---

# fastly_ip_ranges

Use this data source to get the [IP ranges][1] of Fastly edge nodes.

## Example Usage

```hcl
data "fastly_ip_ranges" "fastly" {}

resource "aws_security_group" "from_fastly" {
  name = "from_fastly"

  ingress {
    from_port   = "443"
    to_port     = "443"
    protocol    = "tcp"
    cidr_blocks = ["${data.fastly_ip_ranges.fastly.cidr_blocks}"]
  }
}
```

## Attributes Reference

* `cidr_blocks` - The lexically ordered list of CIDR blocks.

[1]: https://docs.fastly.com/guides/securing-communications/accessing-fastlys-ip-ranges
