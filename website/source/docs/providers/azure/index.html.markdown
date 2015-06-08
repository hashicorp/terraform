---
layout: "azure"
page_title: "Provider: Azure"
sidebar_current: "docs-azure-index"
description: |-
  The Azure provider is used to interact with the many resources supported by Azure. The provider needs to be configured with a publish settings file and optionally a subscription ID before it can be used.
---

# Azure Provider

The Azure provider is used to interact with the many resources supported
by Azure. The provider needs to be configured with a [publish settings
file](https://manage.windowsazure.com/publishsettings) and optionally a
subscription ID before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Azure Provider
provider "azure" {
    settings_file = "${var.azure_settings_file}"
}

# Create a web server
resource "azure_instance" "web" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `settings_file` - (Optional) The path to a publish settings file used to
  authenticate with the Azure API. You can download the settings file here:
  https://manage.windowsazure.com/publishsettings. You must either provide
  (or source from the `AZURE_SETTINGS_FILE` environment variable) a settings
  file or both a `subscription_id` and `certificate`.

* `subscription_id` - (Optional) The subscription ID to use. If a
  `settings_file` is not provided `subscription_id` is required. It can also
  be sourced from the `AZURE_SUBSCRIPTION_ID` environment variable.

* `certificate` - (Optional) The certificate used to authenticate with the
  Azure API. If a `settings_file` is not provided `certificate` is required.
  It can also be sourced from the `AZURE_CERTIFICATE` environment variable.
