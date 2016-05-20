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

## Creating Credentials

Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details).

Using the 'Classic' Portal:

- Select **Active Directory** from the left pane and select the directory you wish to use
- Select **Applications** from the options at the top of the page
- Select **Add** from the bottom of the page. Choose **Add an application my organization is developing**
- Add a friendly name for the application e.g. **Terraform**. Leave **Web Application And/Or Web API** selected and click the arrow for the next page
- Add two valid URIs. These aren't used an can be anything e.g. http://terra.form. Click the arrow to complete the wizard
- You should now be on the page for the application. Click on **Configure** at the top of the page. Scroll down to the middle of the page where you will see the value for `client_id`
- In the **Keys** section of the page, select a suitable duration and click **Save** at the bottom of the page. This will then display the value for `client_secret`. This will disappear once you move off the page
- Click **View Endpoints** at the bottom of the page. This will display a list of URIs. Extract the GUID from the bottom URI for **OAUTH 2.0 AUTHORIZATION ENDPOINT**. This is the `tenant_id`

To enable the application for use with Azure RM, you now need to switch to the 'New' Portal:

- Select **Subscriptions** from the left panel. Select the subscription that you want to use. In the Subscription details pane, click **All Settings** and then **Users**
- Click **Add** and then select an appropriate role for the tasks you want to complete with Terraform. You can find details on the built in roles [here](https://azure.microsoft.com/en-gb/documentation/articles/role-based-access-built-in-roles/)
- Type in the name of the application added in the 'Classic' Portal. You need to type this as it won't be shown in the user list. Click on the appropriate user in the list and then click **Select**
- Click **OK** in the **Add Access** panel. The changes will now be saved   

Microsoft have a more complete guide in the Azure documentation: [Create Active Directory application and service principle](https://azure.microsoft.com/en-us/documentation/articles/resource-group-create-service-principal-portal/)

## Testing

Credentials must be provided via the `ARM_SUBSCRIPTION_ID`, `ARM_CLIENT_ID`,
`ARM_CLIENT_SECRET` and `ARM_TENANT_ID` environment variables in order to run
acceptance tests.
