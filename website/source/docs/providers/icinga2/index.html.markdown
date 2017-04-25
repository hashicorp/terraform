---
layout: "icinga2"
page_title: "Provider: Icinga2"
sidebar_current: "docs-icinga2-index"
description: |-
  The Icinga2 provider is used to configure hosts to be monitored by Icinga2 servers. The provider needs to be configured with the API URL of the Icinga2 server and credentials for an API user with the appropriate permissions.
---


# Icinga2 Provider

The Icinga2 provider is used to configure hosts to be monitored by
[Icinga2](https://www.icinga.com/products/icinga-2/) servers. The provider
needs to be configured with the API URL of the Icinga2 server and credentials
for an API user with the appropriate permissions.

## Example Usage

```hcl
# Configure the Icinga2 provider
provider "icinga2" {
  api_url                  = "https://192.168.33.5:5665/v1"
  api_user                 = "root"
  api_password             = "icinga"
  insecure_skip_tls_verify = true
}

# Configure a host
resouce "icinga2_host" "web-server" {
  # ...
}
```

## Authentication

### Static credentials ###

Static credentials can be provided by adding an `api_user` and `api_password` in-line in the
icinga2 provider block:

Usage:

```hcl
provider "icinga2" {
  api_url      = "https://192.168.33.5:5665/v1"
  api_user     = "root"
  api_password = "icinga"
}
```


###Environment variables

You can provide your credentials via `ICINGA2_API_USER` and `ICINGA2_API_PASSWORD`,
environment variables, storing your Icinga2 API user and password, respectively.
`ICINGA2_API_URL` and `ICINGA2_INSECURE_SKIP_TLS_VERIFY` are also used, if applicable:

```hcl
provider "icinga" {}
```

Usage:

```hcl
$ export ICINGA2_API_URL=https://192.168.33.5:5665/v1
$ export ICINGA2_API_USER=root
$ export ICINGA2_API_PASSWORD=icinga
$ export ICINGA2_INSECURE_SKIP_TLS_VERIFY=true
$ terraform plan
```

## Argument Reference

* ``api_url`` - (Required) The root API URL of an Icinga2 server. May alternatively be
  set via the ``ICINGA2_API_URL`` environment variable.

* ``api_user`` - (Required) The API username to use to
  authenticate to the Icinga2 server. May alternatively
  be set via the ``ICINGA2_API_USER`` environment variable.

* ``api_password`` - (Required) The password to use to
  authenticate to the Icinga2 server. May alternatively
  be set via the ``ICINGA2_API_PASSWORD`` environment variable.

* ``insecure_skip_tls_verify`` - (optional) Defaults to false. If set to true,
  verification of the Icinga2 server's SSL certificate is disabled. This is a security
  risk and should be avoided. May alternatively be set via the
  ``ICINGA2_INSECURE_SKIP_TLS_VERIFY`` environment variable.
