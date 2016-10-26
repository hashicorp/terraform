---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_security_group"
sidebar_current: "docs-cloudstack-resource-security_group"
description: |-
  Creates security group.
---

# cloudstack\_security\_group

Creates security group.

## Example Usage

```
resource "cloudstack_security_group" "default" {
  name = "allow_web"
  description = "Allow access to HTTP and HTTPS"

  rules = [
    {
      cidr_list    = "0.0.0.0/0"
      protocol     = "tcp"
      ports        = "80"
      traffic_type = "ingress"
    },
    {
      cidr_list    = "0.0.0.0/0"
      protocol     = "tcp"
      ports        = "443"
      traffic_type = "ingress"
    },
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The Name of the Security Group to create. Changing this
    forces a new resource to be created.

* `description` - (Optional) The description of the Security Group to create.
    Changing this forces a new resource to be created.

* `rules` - (Optional) List of rule blocks, supported fields documented below.

The `rule` block supports:

* `cidr_list` - (Optional) A CIDR list to allow access to the given ports.

* `security_group` - (Optional) A Security Group to apply the rules to.

* `protocol` - (Required) The name of the protocol to allow. Valid options are:
    `tcp`, `udp` and `icmp`.

* `ports` - (Optional) List of ports and/or port ranges to allow. This can only
    be specified if the protocol is TCP or UDP.

* `traffic_type` - (Optional) Weither Ingress or Egress. (Default: Ingress).

## Attributes Reference

The following attributes are exported:

* `name` - The name of the Security Group.

