---
layout: "datadog"
page_title: "Provider: Datadog"
sidebar_current: "docs-datadog-index"
description: |-
  The Datadog provider is used to interact with the resources supported by Datadog. The provider needs to be configured with the proper credentials before it can be used.
---

# Datadog Provider

The [Datadog](https://www.datadoghq.com) provider is used to interact with the
resources supported by Datadog. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Datadog provider
provider "datadog" {
	api_key = "${var.datadog_api_key}"
	app_key = "${var.datadog_app_key}"
}

# Create a new monitor 
resource "datadog_monitor" "default" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `api_key` - (Required) Datadog API key
* `app_key` - (Required) Datadog APP key

