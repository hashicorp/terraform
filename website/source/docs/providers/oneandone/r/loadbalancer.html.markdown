---
layout: "oneandone"
page_title: "1&1: oneandone_loadbalancer"
sidebar_current: "docs-oneandone-resource-loadbalancer"
description: |-
  Creates and manages 1&1 Load Balancer.
---

# oneandone\_server

Manages a Load Balancer on 1&1

## Example Usage

```hcl
resource "oneandone_loadbalancer" "lb" {
  name = "test_lb"
  method = "ROUND_ROBIN"
  persistence = true
  persistence_time = 60
  health_check_test = "TCP"
  health_check_interval = 300
  datacenter = "GB"
  rules = [
    {
      protocol = "TCP"
      port_balancer = 8080
      port_server = 8089
      source_ip = "0.0.0.0"
    },
    {
      protocol = "TCP"
      port_balancer = 9090
      port_server = 9099
      source_ip = "0.0.0.0"
    }
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the load balancer.
* `description` - (Optional) Description for the load balancer
* `method` - (Required)  Balancing procedure Can be `ROUND_ROBIN` or `LEAST_CONNECTIONS`
* `datacenter` - (Optional) Location of desired 1and1 datacenter. Can be `DE`, `GB`, `US` or `ES`
* `persistence` - (Optional) True/false defines whether persistence should be turned on/off
* `persistence_time` - (Optional) Persistence duration in seconds
* `health_check_test` - (Optional) Can be `TCP` or`ICMP`.
* `health_check_test_interval` - (Optional) 
* `health_check_test_path` - (Optional) 
* `health_check_test_parser` - (Optional) 

Loadbalancer rules (`rules`) support the following

* `protocol` - (Required)  The protocol for the rule. Allowed values are `TCP`, `UDP`, `TCP/UDP`, `ICMP` and `IPSEC`.
* `port_balancer` - (Required) 
* `port_server` - (Required) 
* `source_ip` - (Required) 
