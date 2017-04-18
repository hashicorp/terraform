---
layout: "profitbricks"
page_title: "Provider: ProfitBricks"
sidebar_current: "docs-profitbricks-index"
description: |-
  A provider for ProfitBricks.
---

# ProfitBricks Provider

The ProfitBricks provider gives the ability to deploy and configure resources using ProfitBricks Cloud API.

Use the navigation to the left to read about the available resources.


## Usage

The provider needs to be configured with proper credentials before it can be used.


```hcl
$ export PROFITBRICKS_USERNAME="profitbricks_username"
$ export PROFITBRICKS_PASSWORD="profitbricks_password"
$ export PROFITBRICKS_API_URL="profitbricks_rest_url"
```

Or you can provide your credentials like this:


The credentials provided in `.tf` file will override credentials in the environment variables.

## Example Usage


```hcl
provider "profitbricks" {
  username = "profitbricks_username"
  password = "profitbricks_password"
  endpoint = "profitbricks_rest_url"
  retries  = 100
}

resource "profitbricks_datacenter" "main" {
  # ...
}
```


## Configuration Reference

The following arguments are supported:

* `username` - (Required) If omitted, the `PROFITBRICKS_USERNAME` environment variable is used.

* `password` - (Required) If omitted, the `PROFITBRICKS_PASSWORD` environment variable is used.

* `endpoint` - (Required) If omitted, the `PROFITBRICKS_API_URL` environment variable is used.

* `retries` - (Optional) Number of retries while waiting for a resource to be provisioned. Default value is 50.


#Support
You are welcome to contact us with questions or comments at [ProfitBricks DevOps Central](https://devops.profitbricks.com/).