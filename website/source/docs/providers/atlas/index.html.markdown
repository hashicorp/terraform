---
layout: "atlas"
page_title: "Provider: Atlas"
sidebar_current: "docs-atlas-index"
description: |-
  The Atlas provider is used to interact with configuration,
  artifacts, and metadata managed by the Atlas service.
---

# Atlas Provider

The Atlas provider is used to interact with resources, configuration,
artifacts, and metadata managed by [Atlas](https://atlas.hashicorp.com).
The provider needs to be configured with the proper credentials before
it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Atlas provider
provider "atlas" {
    token = "${var.atlas_token}"
}

# Fetch an artifact configuration
resource "atlas_artifact" "web" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Optional) Atlas server endpoint. Defaults to public Atlas.
  This is only required when using an on-premise deployment of Atlas. This can
  also be specified with the `ATLAS_ADDRESS` shell environment variable.

* `token` - (Required) API token. This can also be specified with the
  `ATLAS_TOKEN` shell environment variable.

