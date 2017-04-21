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

```
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

* `name` - (Required) [String] The name of the load balancer.
* `description` - (Optional) [String] Description for the load balancer
* `method` - (Required) [String] Balancing procedure ["ROUND_ROBIN", "LEAST_CONNECTIONS"]
* `datacenter` - (Optional) [String]  Location of desired 1and1 datacenter ["DE", "GB", "US", "ES" ]
* `persistence` - (Optional) [Boolean]  True/false defines whether persistence should be turned on/off
* `persistence_time` - (Optional) [Integer] Persistance duration in seconds
* `health_check_test` - (Optional) [String] ["TCP", "ICMP"]
* `health_check_test_interval` - (Optional) [String]
* `health_check_test_path` - (Optional) [String]
* `health_check_test_parser` - (Optional) [String]

Loadbalancer rules (`rules`) support the following

* `protocol` - (Required) [String]  The protocol for the rule ["TCP", "UDP", "TCP/UDP", "ICMP", "IPSEC"]
* `port_balancer` - (Required) [String]
* `port_server` - (Required) [String]
* `source_ip` - (Required) [String]
