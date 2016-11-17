---
layout: "akamai"
page_title: "Provider: Akamai"
sidebar_current: "docs-akamai-index"
description: |-
  The Akamai provider is used to manage Akamai GTM configuration.
---

# Akamai Provider

The Akamai provider is used to manage [Akamai GTM configuration](https://www.akamai.com/us/en/solutions/products/web-performance/global-traffic-management.jsp).

Use the navigation to the left to read about the available resources.

## Example Usage

```
resource "akamai_gtm_domain" "some_domain" {
    name = "some-domain.akadns.net"
    type = "basic"
}

resource "akamai_gtm_data_center" "dc1" {
    name = "dc1"
    domain = "${akamai_gtm_domain.some_domain.name}"
    country = "GB"
    continent = "EU"
    city = "Downpatrick"
    longitude = -5.582
    latitude = 54.367
    depends_on = [
        "akamai_gtm_domain.some_domain"
    ]
}

resource "akamai_gtm_data_center" "dc2" {
    name = "dc2"
    domain = "${akamai_gtm_domain.some_domain.name}"
    country = "IS"
    continent = "EU"
    city = "Snæfellsjökull"
    longitude = -23.776
    latitude = 64.808
    depends_on = [
        "akamai_gtm_data_center.dc1"
    ]
}

resource "akamai_gtm_property" "some_property" {
  domain = "${akamai_gtm_domain.some_domain.name}"
  type = "weighted-round-robin"
  name = "some_property"
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
      name = "health check"
      test_object = "/status"
      test_object_protocol = "HTTP"
      test_interval = 60
      disable_nonstandard_port_warning = false
      http_error_4xx = true
      httpError3xx = true
      httpError5xx = true
      testObjectPort = 80
      testTimeout = 25
  }
  traffic_target {
      enabled = true
      data_center_id = "${akamai_gtm_data_center.dc1.id}"
      weight = 50.0
      name = "${akamai_gtm_data_center.dc1.name}"
      servers = [
          "1.2.3.4",
          "1.2.3.5"
      ]
  }
  traffic_target {
      enabled = true
      data_center_id = "${akamai_gtm_data_center.dc2.id}"
      weight = 50.0
      name = "${akamai_gtm_data_center.dc2.name}"
      servers = [
          "1.2.3.6",
          "1.2.3.7"
      ]
  }
}
```
