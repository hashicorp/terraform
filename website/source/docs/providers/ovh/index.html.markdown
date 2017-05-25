---
layout: "ovh"
page_title: "Provider: OVH"
sidebar_current: "docs-ovh-index"
description: |-
  The OVH provider is used to interact with the many resources supported by OVH. The provider needs to be configured with the proper credentials before it can be used.
---

# OVH Provider

The OVH provider is used to interact with the
many resources supported by OVH. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the OVH Provider
provider "ovh" {
  endpoint           = "ovh-eu"
  application_key    = "yyyyyy"
  application_secret = "xxxxxxxxxxxxxx"
  consumer_key       = "zzzzzzzzzzzzzz"
}

# Create a public cloud user
resource "ovh_publiccloud_user" "user-test" {
  # ...
}
```

## Configuration Reference

The following arguments are supported:

* `endpoint` - (Required) Specify which API  endpoint to use.
  It can be set using the OVH_ENDPOINT environment
  variable. Value can be set to either "ovh-eu" or "ovh-ca".

* `application_key` - (Required) The API Application Key. If omitted,
  the `OVH_APPLICATION_KEY` environment variable is used.

* `application_secret` - (Required) The API Application Secret. If omitted,
  the `OVH_APPLICATION_SECRET` environment variable is used.

* `consumer_key` - (Required) The API Consumer key. If omitted,
  the `OVH_CONSUMER_KEY` environment variable is used.


## Testing and Development

In order to run the Acceptance Tests for development, the following environment
variables must also be set:

* `OVH_VRACK` - The id of the vrack to use.

* `OVH_PUBLIC_CLOUD` - The id of the public cloud project.

You should be able to use any OVH environment to develop on as long as the
above environment variables are set.
