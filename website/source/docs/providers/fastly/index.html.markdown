---
layout: "fastly"
page_title: "Provider: Fastly"
sidebar_current: "docs-fastly-index"
description: |-
  Fastly
---

# Fastly Provider

The Fastly provider is used to interact with the content delivery network (CDN)
provided by Fastly.

In order to use this Provider, you must have an active account with Fastly.
Pricing and signup information can be found at https://www.fastly.com/signup

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Fastly Provider
provider "fastly" {
  api_key = "test"
}

# Create a Service
resource "fastly_service_v1" "myservice" {
  name = "myawesometestservice"

  # ...
}
```

## Authentication

The Fastly provider offers an API key based method of providing credentials for
authentication. The following methods are supported, in this order, and
explained below:

- Static API key
- Environment variables


### Static API Key ###

Static credentials can be provided by adding a `api_key` in-line in the
Fastly provider block:

Usage:

```hcl
provider "fastly" {
  api_key = "test"
}

resource "fastly_service_v1" "myservice" {
  # ...
}
```

The API key for an account can be found on the Account page: https://app.fastly.com/#account

###Environment variables

You can provide your API key via `FASTLY_API_KEY` environment variable,
representing your Fastly API key. When using this method, you may omit the
Fastly `provider` block entirely:

```hcl
resource "fastly_service_v1" "myservice" {
  # ...
}
```

Usage:

```
$ export FASTLY_API_KEY="afastlyapikey"
$ terraform plan
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `api_key` - (Optional) This is the API key. It must be provided, but
  it can also be sourced from the `FASTLY_API_KEY` environment variable
