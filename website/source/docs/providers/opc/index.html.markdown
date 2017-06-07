---
layout: "opc"
page_title: "Provider: Oracle Public Cloud"
sidebar_current: "docs-opc-index"
description: |-
  The Oracle Public Cloud provider is used to interact with the many resources supported by the Oracle Public Cloud. The provider needs to be configured with credentials for the Oracle Public Cloud API.
---

# Oracle Public Cloud Provider

The Oracle Public Cloud provider is used to interact with the many resources supported by the Oracle Public Cloud. The provider needs to be configured with credentials for the Oracle Public Cloud API.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Oracle Public Cloud
provider "opc" {
  user            = "..."
  password        = "..."
  identity_domain = "..."
  endpoint        = "..."
}

# Create an IP Reservation
resource "opc_compute_ip_reservation" "production" {
  parent_pool = "/oracle/public/ippool"
  permanent = true
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Optional) The username to use, generally your email address. It can also
  be sourced from the `OPC_USERNAME` environment variable.

* `password` - (Optional) The password associated with the username to use. It can also be sourced from
  the `OPC_PASSWORD` environment variable.

* `identity_domain` - (Optional) The identity domain to use. It can also be sourced from
  the `OPC_IDENTITY_DOMAIN` environment variable.

* `endpoint` - (Optional) The API endpoint to use, associated with your Oracle Public Cloud account. This is known as the `REST Endpoint` within the Oracle portal. It can also be sourced from the `OPC_ENDPOINT` environment variable.

* `max_retries` - (Optional) The maximum number of tries to make for a successful response when operating on resources within Oracle Public Cloud. It can also be sourced from the `OPC_MAX_RETRIES` environment variable. Defaults to 1.

* `insecure` - (Optional) Skips TLS Verification for using self-signed certificates. Should only be used if absolutely needed. Can also via setting the `OPC_INSECURE` environment variable to `true`.

## Testing

Credentials must be provided via the `OPC_USERNAME`, `OPC_PASSWORD`,
`OPC_IDENTITY_DOMAIN` and `OPC_ENDPOINT` environment variables in order to run
acceptance tests.
