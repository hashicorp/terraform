---
layout: "runscope"
page_title: "Provider: Runscope"
sidebar_current: "docs-runscope-index"
description: |-
  The Runscope provider is used to interact with the resources supported by Runscope. The provider needs to be configured with the proper access token before it can be used.
---

# Runscope Provider

The Runscope provider is used to interact with the
resources supported by Runscope. The provider needs to be configured
with an access token before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Runscope provider
provider "runscope" {
  access_token = "${var.access_token}"
}

# Create a bucket
resource "runscope_bucket" "main" {
  name         = "terraform-ftw"
  team_uuid    = "870ed937-bc6e-4d8b-a9a5-d7f9f2412fa3"
}

# Create a test in the bucket
resource "runscope_test" "api" {
  name         = "api-test"
  description  = "checks the api is up and running"
  bucket_id    = "${runscope_bucket.main}"
}
```

## Argument Reference

The following arguments are supported:

* `access_token` - (Required) The Runscope access token.
  This can also be specified with the `RUNSCOPE_ACCESS_TOKEN` shell
  environment variable.
* `api_url` - (Optional) If set, specifies the Runscope api url, this
   defaults to `"https://api.runscope.com`. This can also be specified
   with the `RUNSCOPE_API_URL` shell environment variable.
