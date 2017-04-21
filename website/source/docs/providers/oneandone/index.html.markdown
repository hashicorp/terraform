---
layout: "oneandone"
page_title: "Provider: 1&1"
sidebar_current: "docs-oneandone-index"
description: |-
  A provider for 1&1.
---

# 1&1 Provider

The 1&1 provider gives the ability to deploy and configure resources using the 1&1 Cloud Server API.

Use the navigation to the left to read about the available resources.


## Usage

The provider needs to be configured with proper credentials before it can be used.


```text
$ export ONEANDONE_TOKEN="oneandone_token"
```

Or you can provide your credentials like this:


The credentials provided in `.tf` file will override credentials in the environment variables.

## Example Usage


```hcl
provider "oneandone"{
  token = "oneandone_token"
  endpoint = "oneandone_endpoint"
  retries = 100
}

resource "oneandone_server" "server" {
  # ...
}
```


## Configuration Reference

The following arguments are supported:

* `token` - (Required) If omitted, the `ONEANDONE_TOKEN` environment variable is used.

* `endpoint` - (Optional)

* `retries` - (Optional) Number of retries while waiting for a resource to be provisioned. Default value is 50.
