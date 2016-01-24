---
layout: "azurerm"
page_title: "Provider: Azure Resource Manager"
sidebar_current: "docs-azurerm-index"
description: |-
  The Azure Resource Manager provider is used to interact with the many resources supported by Azure, via the ARM API. This supercedes the Azure provider, which interacts with Azure using the Service Management API. The provider needs to be configured with a credentials file, or credentials needed to generate OAuth tokens for the ARM API.
---

# Azure Resource Manager Provider

The Azure Resource Manager provider is used to interact with the many resources
supported by Azure, via the ARM API. This supercedes the Azure provider, which
interacts with Azure using the Service Management API. The provider needs to be
configured with the credentials needed to generate OAuth tokens for the ARM API.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Azure Resource Manager Provider
provider "azurerm" {
  subscription_id = "..."
  client_id       = "..."
  client_secret   = "..."
  tenant_id       = "..."
}

# Create a resource group
resource "azurerm_resource_group" "production" {
    name     = "production"
    location = "West US"
}

# Create a virtual network in the web_servers resource group
resource "azurerm_virtual_network" "network" {
  name                = "productionNetwork"
  address_space       = ["10.0.0.0/16"]
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.production.name}"

  subnet {
    name           = "subnet1"
    address_prefix = "10.0.1.0/24"
  }

  subnet {
    name           = "subnet2"
    address_prefix = "10.0.2.0/24"
  }

  subnet {
    name           = "subnet3"
    address_prefix = "10.0.3.0/24"
  }
}

```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Optional) The subscription ID to use. It can also
  be sourced from the `ARM_SUBSCRIPTION_ID` environment variable.

* `client_id` - (Optional) The client ID to use. It can also be sourced from
  the `ARM_CLIENT_ID` environment variable.

* `client_secret` - (Optional) The client secret to use. It can also be sourced from
  the `ARM_CLIENT_SECRET` environment variable.

* `tenant_id` - (Optional) The tenant ID to use. It can also be sourced from the
  `ARM_TENANT_ID` environment variable.

## Testing:

Credentials must be provided via the `ARM_SUBSCRIPTION_ID`, `ARM_CLIENT_ID`,
`ARM_CLIENT_SECRET` and `ARM_TENANT_ID` environment variables in order to run
acceptance tests.
