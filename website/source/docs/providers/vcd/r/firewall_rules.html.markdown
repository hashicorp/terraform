---
layout: "vcd"
page_title: "vCloudDirector: vcd_firewall_rules"
sidebar_current: "docs-vcd-resource-firewall-rules"
description: |-
  Provides a vCloud Director Firewall resource. This can be used to create, modify, and delete firewall settings and rules.
---

# vcd\_firewall\_rules

Provides a vCloud Director Firewall resource. This can be used to create,
modify, and delete firewall settings and rules.

## Example Usage

```hcl
resource "vcd_firewall_rules" "fw" {
  edge_gateway   = "Edge Gateway Name"
  default_action = "drop"

  rule {
    description      = "deny-ftp-out"
    policy           = "deny"
    protocol         = "tcp"
    destination_port = "21"
    destination_ip   = "any"
    source_port      = "any"
    source_ip        = "10.10.0.0/24"
  }

  rule {
    description      = "allow-outbound"
    policy           = "allow"
    protocol         = "any"
    destination_port = "any"
    destination_ip   = "any"
    source_port      = "any"
    source_ip        = "10.10.0.0/24"
  }
}

resource "vcd_vapp" "web" {
  # ...
}

resource "vcd_firewall_rules" "fw-web" {
  edge_gateway   = "Edge Gateway Name"
  default_action = "drop"

  rule {
    description      = "allow-web"
    policy           = "allow"
    protocol         = "tcp"
    destination_port = "80"
    destination_ip   = "${vcd_vapp.web.ip}"
    source_port      = "any"
    source_ip        = "any"
  }
}
```

## Argument Reference

The following arguments are supported:

* `edge_gateway` - (Required) The name of the edge gateway on which to apply the Firewall Rules
* `default_action` - (Required) Either "allow" or "deny". Specifies what to do should none of the rules match
* `rule` - (Optional) Configures a firewall rule; see [Rules](#rules) below for details.

<a id="rules"></a>
## Rules

Each firewall rule supports the following attributes:

* `description` - (Required) Description of the fireall rule
* `policy` - (Required) Specifies what to do when this rule is matched. Either "allow" or "deny"
* `protocol` - (Required) The protocol to match. One of "tcp", "udp", "icmp" or "any"
* `destination_port` - (Required) The destination port to match. Either a port number or "any"
* `destination_ip` - (Required) The destination IP to match. Either an IP address, IP range or "any"
* `source_port` - (Required) The source port to match. Either a port number or "any"
* `source_ip` - (Required) The source IP to match. Either an IP address, IP range or "any"
