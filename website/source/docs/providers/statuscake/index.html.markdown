---
layout: "statuscake"
page_title: "Provider: StatusCake"
sidebar_current: "docs-statuscake-index"
description: |-
  The StatusCake provider configures tests in StatusCake.
---

# StatusCake Provider

The StatusCake provider allows Terraform to create and configure tests in [StatusCake](https://www.statuscake.com/). StatusCake is a tool that helps to
monitor the uptime of your service via a network of monitoring centers throughout the world

The provider configuration block accepts the following arguments:

* ``username`` - (Required) The username for the statuscake account. May alternatively be set via the
  ``STATUSCAKE_USERNAME`` environment variable.

* ``apikey`` - (Required) The API auth token to use when making requests. May alternatively
  be set via the ``STATUSCAKE_APIKEY`` environment variable.
  
Use the navigation to the left to read about the available resources.

## Example Usage

```
provider "statuscake" {
    username = "testuser"
    apikey = "12345ddfnakn"
}

resource "statuscake_test" "google" {
    website_name = "google.com"
    website_url = "www.google.com"
    test_type = "HTTP"
    check_rate = 300
}
```
