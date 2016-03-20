---
layout: "triton"
page_title: "Triton: triton_firewall_rule"
sidebar_current: "docs-triton-firewall"
description: |-
    The `triton_firewall_rule` resource represents a rule for the Triton cloud firewall. 
---

# triton\_firewall\_rule

The `triton_firewall_rule` resource represents a rule for the Triton cloud firewall.

## Example Usages

Allow traffic on ports tcp/80 and tcp/443 to machines with the 'www' tag from any source


```
resource "triton_firewall_rule" "www" {
    rule = "FROM any TO tag www ALLOW tcp (PORT 80 AND PORT 443)"
    enabled = true
}
                
```
Block traffic on port tcp/143 to all machines


```
resource "triton_firewall_rule" "imap" {
    rule = "FROM any TO all vms BLOCK tcp port 143"
    enabled = true
}
                
```

## Argument Reference

The following arguments are supported:

* `rule` - (string, Required)
    The firewall rule described using the Cloud API rule syntax defined at https://docs.joyent.com/public-cloud/network/firewall/cloud-firewall-rules-reference.

* `enabled` - (boolean)  Default: `false`
    Whether the rule should be effective.

## Attribute Reference

The following attributes are exported:

* `id` - (string) - The identifier representing the firewall rule in Triton. 
