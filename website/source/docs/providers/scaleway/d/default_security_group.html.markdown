---
layout: "scaleway"
page_title: "Scaleway: default_security_group"
sidebar_current: "docs-scaleway-datasource-default_security_group"
description: |-
    Lookup the default security group
---

# scaleway\_default\_security\_group

The Default Security Group data source allows identifying the default security group
which is created by Scaleway for you. This is usefull when you want to attach additional
security group rules

## Example Usage

```
# Declare the data source
data "scaleway_default_security_group" "default" {}

# reference default security group
resource "scaleway_security_group_rule" "http" {
  security_group = "${data.scaleway_default_security_group.default.id}"

  action = "drop"
  direction = "inbound"
  ip_range = "0.0.0.0/0"
  protocol = "TCP"
  port = 80
}
```

## Argument Reference

There are no arguments for this data source.

## Attributes Reference

The following attributes are exported:

* `id` - the ID of the default security group.
* `name` - the name of the default security group.
* `description` - the description of the default security group.
