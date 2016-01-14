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
  settings_file = "${file("credentials.publishsettings")}"
}

# Create a web server
resource "azure_instance" "web" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `publish_settings` - (Optional) Contents of a valid `publishsettings` file,
  used to authenticate with the Azure API. You can download the settings file
  here: https://manage.windowsazure.com/publishsettings. You must either
  provide publish settings or both a `subscription_id` and `certificate`. It
  can also be sourced from the `AZURE_PUBLISH_SETTINGS` environment variable.

* `subscription_id` - (Optional) The subscription ID to use. If a
  `settings_file` is not provided `subscription_id` is required. It can also
  be sourced from the `AZURE_SUBSCRIPTION_ID` environment variable.

* `certificate` - (Optional) The certificate used to authenticate with the
  Azure API. If a `settings_file` is not provided `certificate` is required.
  It can also be sourced from the `AZURE_CERTIFICATE` environment variable.

These arguments are supported for backwards compatibility, and may be removed
in a future version:

* `settings_file` - __Deprecated: please use `publish_settings` instead.__
  Path to or contents of a valid `publishsettings` file, used to
  authenticate with the Azure API. You can download the settings file here:
  https://manage.windowsazure.com/publishsettings. You must either provide
  (or source from the `AZURE_SETTINGS_FILE` environment variable) a settings
  file or both a `subscription_id` and `certificate`.

## Testing:

The following environment variables must be set for the running of the
acceptance test suite:

* A valid combination of the above which are required for authentification.

* `AZURE_STORAGE` - The name of a storage account to be used in tests which
  require a storage backend. The storage account needs to be located in
  the Western US Azure region.
