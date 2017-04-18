---
layout: "grafana"
page_title: "Provider: Grafana"
sidebar_current: "docs-grafana-index"
description: |-
  The Grafana provider configures data sources and dashboards in Grafana.
---

# Grafana Provider

The Grafana provider configures data sources and dashboards in
[Grafana](http://grafana.org/), which is a web application for creating,
viewing and sharing metrics dashboards.

The provider configuration block accepts the following arguments:

* ``url`` - (Required) The root URL of a Grafana server. May alternatively be
  set via the ``GRAFANA_URL`` environment variable.

* ``auth`` - (Required) The API token or username/password to use to
  authenticate to the Grafana server. If username/password is used, they
  are provided in a single string and separated by a colon. May alternatively
  be set via the ``GRAFANA_AUTH`` environment variable.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "grafana" {
  url  = "http://grafana.example.com/"
  auth = "1234abcd"
}

resource "grafana_dashboard" "metrics" {
  config_json = "${file("grafana-dashboard.json")}"
}
```
