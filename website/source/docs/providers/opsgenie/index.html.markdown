---
layout: "opsgenie"
page_title: "Provider: OpsGenie"
sidebar_current: "docs-opsgenie-index"
description: |-
  The OpsGenie provider is used to interact with the many resources supported by OpsGenie. The provider needs to be configured with the proper credentials before it can be used.
---

# OpsGenie Provider

The OpsGenie provider is used to interact with the
many resources supported by OpsGenie. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the OpenStack Provider
provider "opsgenie" {
  api_key = "key"
}

# Create a user
resource "opsgenie_user" "test" {
  # ...
}
```

## Configuration Reference

The following arguments are supported:

* `api_key` - (Required) The API Key for the OpsGenie Integration. If omitted, the
  `OPSGENIE_API_KEY` environment variable is used.

You can generate an API Key within OpsGenie by creating a new API Integration with Read/Write permissions.

## Testing and Development

In order to run the Acceptance Tests for development, the following environment
variables must also be set:

* `OPSGENIE_API_KEY` - The API Key used for the OpsGenie Integration.
