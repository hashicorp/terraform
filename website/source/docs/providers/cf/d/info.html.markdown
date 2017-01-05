---
layout: "cf"
page_title: "Cloud Foundry: cf_info"
sidebar_current: "docs-cf-datasource-info"
description: |-
  Get information on a Cloud Foundry target.
---

# cf\_info

Gets information on a Cloud Foundry target.

## Example Usage

```
data "cf_info" "info" {}
```

## Attributes Reference

The following attributes are exported:

* `api_version` - The Cloud Foundry API version
* `auth_endpoint` - The autentication endpoint URL
* `uaa_endpoint` - The UAA endpoint URL
* `routing_endpoint` - The routing endpoint URL
* `logging_endpoint` - The endpoint for logging services
* `doppler_endpoint` - The endpoint for doppler services
