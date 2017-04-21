---
layout: "mailgun"
page_title: "Provider: Mailgun"
sidebar_current: "docs-mailgun-index"
description: |-
  The Mailgun provider is used to interact with the resources supported by Mailgun. The provider needs to be configured with the proper credentials before it can be used.
---

# Mailgun Provider

The Mailgun provider is used to interact with the
resources supported by Mailgun. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Mailgun provider
provider "mailgun" {
  api_key = "${var.mailgun_api_key}"
}

# Create a new domain
resource "mailgun_domain" "default" {
  # ...
}
```

## Argument Reference

The following arguments are supported:

* `api_key` - (Required) Mailgun API key

