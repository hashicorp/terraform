---
layout: "azurerm"
page_title: "Provider: Azure Resource Manager"
sidebar_current: "docs-azurerm-index"
description: |-
  The Azure Resource Manager provider is used to interact with the many resources supported by Azure, via the ARM API. This supercedes the Azure provider, which interacts with Azure using the Service Management API. The provider needs to be configured with a credentials file, or credentials needed to generate OAuth tokens for the ARM API.
---

# Microsoft Azure Provider

The Microsoft Azure provider is used to interact with the many
resources supported by Azure, via the ARM API. This supercedes the [legacy Azure
provider][asm], which interacts with Azure using the Service Management API. The
provider needs to be configured with the credentials needed to generate OAuth
tokens for the ARM API.

[asm]: /docs/providers/azure/index.html

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Microsoft Azure Provider
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

There are two high-level tasks to complete.  The first is to create an App Registration with Azure Active Directory.  You can do this in either the New ARM portal (http://portal.azure.com) or the older 'Classic' portal (http://manage.windowsazure.com).

The second task is to grant permissions for the Application Registration in your Subscription.

To create the App Registration using the New ARM portal:

- Select **Azure Active Directory** from the left pane
- Select the **App Registrations** tile from the Overview Section
- Click **Endpoints** at the top of the App Registrations blade.  This will display a list of URIs. Extract the GUID from the bottom URI for **OAUTH 2.0 AUTHORIZATION ENDPOINT**. This is the `tenant_id`
- Select **Add** from the top of the blade.
- Add a friendly name for the application e.g. **Terraform**. Choose **Web App / API** for Application Type
- Add a valid URI as the Sign-on URL. This isn't used and can be anything e.g. http://terra.form.
- Click **Create** at the bottom to create the App Registration
- Choose your new App Registration to show details
- You should now be on the blade for your App Registration.  At the top, notice the "Application ID" GUID.  You'll use this as the `client_id`
- If the Settings blade for your Application Registration is not showing, click on **All Settings**
- Click on **Keys**. Enter a name for your key in **Key description** and choose an expiration duration.  When you click **Save** at the top of the blade, the key value will be displayed.  Once it is displayed, you then use this as the value for `client_secret`. This will disappear once you move off the page
- Click **Required Permissions**.  Click **Add**.  This will allow us to add permission to use the Windows Azure Service Management API to the App Registration.  On Step 1, choose Windows Azure Service Management API.  Click **Select**.  On Step 2, check the box next to "Access Azure Service Management as organization users".  Click **Select**.  Click **Done** to finish adding the permission.

To create the App Reigstration using the 'Classic' portal:

- Select **Active Directory** from the left pane and select the directory you wish to use
- Select **Applications** from the options at the top of the page
- Select **Add** from the bottom of the page. Choose **Add an application my organization is developing**
- Add a friendly name for the application e.g. **Terraform**. Leave **Web Application And/Or Web API** selected and click the arrow for the next page
- Add two valid URIs. These aren't used an can be anything e.g. http://terra.form. Click the arrow to complete the wizard
- You should now be on the page for the application. Click on **Configure** at the top of the page. Scroll down to the middle of the page where you will see the value for `client_id`
- In the **Keys** section of the page, select a suitable duration and click **Save** at the bottom of the page. This will then display the value for `client_secret`. This will disappear once you move off the page
- Click **View Endpoints** at the bottom of the page. This will display a list of URIs. Extract the GUID from the bottom URI for **OAUTH 2.0 AUTHORIZATION ENDPOINT**. This is the `tenant_id`

To grant permissions to the App Registration to your subscription, you now must to use to the 'ARM' Portal:

- Select **Subscriptions** from the left panel. Select the subscription that you want to use. In the Subscription details pane, click **Access Control (IAM)**
- Click **Add**.  For Step 1 select an appropriate role for the tasks you want to complete with Terraform. You can find details on the built in roles [here](https://azure.microsoft.com/en-gb/documentation/articles/role-based-access-built-in-roles/)
- Type in the name of the application added in the search box. You need to type this as it won't be shown in the user list. Click on the appropriate user in the list and then click **Select**
- Click **OK** in the **Add Access** panel. The changes will now be saved   

Microsoft have a more complete guide in the Azure documentation: [Create Active Directory application and service principle](https://azure.microsoft.com/en-us/documentation/articles/resource-group-create-service-principal-portal/)

## Testing

Credentials must be provided via the `ARM_SUBSCRIPTION_ID`, `ARM_CLIENT_ID`,
`ARM_CLIENT_SECRET` and `ARM_TENANT_ID` environment variables in order to run
acceptance tests.
