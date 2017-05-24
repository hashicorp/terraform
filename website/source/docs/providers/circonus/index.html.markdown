---
layout: "circonus"
page_title: "Provider: Circonus"
sidebar_current: "docs-circonus-index"
description: |-
  A provider for Circonus.
---

# Circonus Provider

The Circonus provider gives the ability to manage a Circonus account.

Use the navigation to the left to read about the available resources.

## Usage

```hcl
provider "circonus" {
  key = "b8fec159-f9e5-4fe6-ad2c-dc1ec6751586"
}
```

## Argument Reference

The following arguments are supported:

* `key` - (Required) The Circonus API Key.
* `api_url` - (Optional) The API URL to use to talk with. The default is `https://api.circonus.com/v2`.
