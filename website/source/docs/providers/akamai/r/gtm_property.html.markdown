---
layout: "akamai"
page_title: "Akamai: akamai_gtm_property"
sidebar_current: "docs-akamai-resource-gtm-property"
description: |-
  Provides access to a GTM property managed by Akamai.
---

# akamai\_gtm\_property

Provides access to a GTM property managed by Akamai.

## Example Usage

```
resource "akamai_gtm_property" "test_property" {
    domain = "some-domain.akadns.net"
    type = "weighted-round-robin"
    name = "test_property"
    balance_by_download_score = false
    dynamic_ttl = 300
    failover_delay = 0
    failback_delay = 0
    handout_mode = "normal"
    health_threshold = 0
    health_max = 0
    health_multiplier = 0
    load_imbalance_percentage = 10
    ipv6 = false
    score_aggregation_type = "mean"
    static_ttl = 600
    stickiness_bonus_percentage = 50
    stickiness_bonus_constant = 0
    use_computed_targets = false
    liveness_test {
        name = "healthcheck"
        test_object = "/status"
        test_object_protocol = "HTTP"
        test_interval = 60
        disable_nonstandard_port_warning = false
        http_error_4xx = true
        http_error_3xx = true
        http_error_5xx = true
        test_object_port = 80
        test_timeout = 25
    }
    traffic_target {
        enabled = true
        datacenter_id = "123"
        weight = 50.0
        name = "traffic_target1"
        servers = [
            "1.2.3.4",
            "1.2.3.5"
        ]
    }
    traffic_target {
        enabled = true
        datacenter_id = "456"
        weight = 50.0
        name = "traffic_target2"
        servers = [
            "1.2.3.6",
            "1.2.3.7"
        ]
    }
}
```

## Argument Reference

The following arguments are supported:

* `domain` - (Required) The Akamai GTM domain with which to associate the GTM property.

* `name` - (Required) A name for the Akamai GTM property. A property is a subdomain within a GTM domain. It is combined with the domain name and is used for load balancing.

* `type` - (Required) Specifies the type of load balancing behavior for this property. Valid values are `failover`, `geographic`, `cidrmapping`, `weighted-round-robin`, `weighted-hashed`, `weighted-round-robin-load-feedback`, `qtr`, or `performance`.

* `score_aggregation_type` - (Required) Specifies how GTM aggregates liveness test scores across different tests when multiple tests are configured. Valid valus are `mean`, `median`, `best`, or `worst`.

* `traffic_target` - (Required) An option for where to direct traffic.

* `handout_mode` - (Required) Relevant when more than one server IP exists in a given datacenter. Specifies the behavior of how IPs are returned when multiple IPs are alive and available. Valid values are `normal`, `persistent`, `one-ip`, `one-ip-hashed`, or `all-live-ips`.

* `balance_by_download_score` - (Optional) Enables download score based load balancing.

* `dynamic_ttl` - (Optional) The TTL for records that might change from moment to moment based on liveness and load balancing.

* `failover_delay` - (Optional) Specifies the desired duration between liveness test failure and when GTM should consider the server down given persistant errors.

* `failback_delay` - (Optional) Specifies the desired duration between liveness test recovery and when GTM should consider the server up given persistant liveness test success.

* `health_threshold` - (Optional) Specifies a threshold value. A server with a score beyond the threshold will not receive traffic.

* `health_max` - (Optional) Specifies an absolute limit beyond which all IPs are considered unhealthy if a `backup_cname` is provided.

* `health_multiplier` - (Optional) Specifies a cutoff value that is computed from the median scores. Any server with a score beyond the cutoff value are considered unhealthy and won't receive traffice.

* `load_imbalance_percentage` - (Optional) Controls the extent to which GTM allows imbalanced load.

* `ipv6` - (Optional) Specifies whether the type of IP addresses handed out by the property are IPv6.

* `static_ttl` - (Optional) Specifies the TTL for record types that do not change moment-to-moment.

* `stickiness_bonus_percentage` - (Optional) Used with `stickiness_bonus_constant` to control datacenter affinity. Specifies that a user should not be switched unless the resulting improvement to their score exceeds a configured threshold.

* `stickiness_bonus_constant` - (Optional) Used with `stickiness_bonus_percentage` to control datacenter affinity. Specifies that a user should not be switched unless the resulting improvement to their score exceeds a configured threshold.

* `use_computed_targets` - (Optional) Specifies whether GTM should automatically compute target load.
