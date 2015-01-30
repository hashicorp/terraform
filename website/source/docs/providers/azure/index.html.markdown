---
layout: "azure"
page_title: "Provider: Microsoft Azure"
sidebar_current: "docs-azure-index"
description: |-
  The Azure provider is used to interact with Microsoft Azure services. The provider needs to be configured with the proper credentials before it can be used.
---

# Azure Provider

The Azure provider is used to interact with
[Microsoft Azure](http://azure.microsoft.com/). The provider needs
to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Azure provider
provider "azure" {
    publish_settings_file = "account.publishsettings"
}

# Create a new instance
resource "azure_virtual_machine" "default" {
    ...
}
```

## Argument Reference

The following keys can be used to configure the provider.

* `publish_settings_file` - (Required) Path to the JSON file used to describe
  your account settings, downloaded from Microsoft Azure. It must be provided,
  but it can also be sourced from the AZURE_PUBLISH_SETTINGS_FILE environment variable.
