---
layout: "azurerm"
page_title: "Provider: Azure Resource Manager"
sidebar_current: "docs-azurerm-index"
description: |-
  The Azure Resource Manager provider is used to interact with the many resources supported by Azure, via the ARM API. This supersedes the Azure provider, which interacts with Azure using the Service Management API. The provider needs to be configured with a credentials file, or credentials needed to generate OAuth tokens for the ARM API.
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

```hcl
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

* `environment` - (Optional) The cloud environment to use. It can also be sourced
  from the `ARM_ENVIRONMENT` environment variable. Supported values are:
  * `public` (default)
  * `usgovernment`
  * `german`
  * `china`

* `skip_provider_registration` - (Optional) Prevents the provider from registering
  the ARM provider namespaces, this can be used if you don't wish to give the Active
  Directory Application permission to register resource providers. It can also be
  sourced from the `ARM_SKIP_PROVIDER_REGISTRATION` environment variable, defaults
  to `false`.

## Creating Credentials

Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details).

It's possible to complete this task in either the [Azure CLI](#creating-credentials-using-the-azure-cli) or in the [Azure Portal](#creating-credentials-in-the-azure-portal) - in both we'll create a Service Principal which has `Contributor` rights to the subscription. [It's also possible to assign other rights](https://azure.microsoft.com/en-gb/documentation/articles/role-based-access-built-in-roles/) depending on your configuration.

### Creating Credentials using the Azure CLI

~> **Note**: if you're using the **China**, **German** or **Government** Azure Clouds - you'll need to first configure the Azure CLI to work with that Cloud.  You can do this by running:

```
$ az cloud set --name AzureChinaCloud|AzureGermanCloud|AzureUSGovernment
```

---

Firstly, login to the Azure CLI using:

```shell
$ az login
```


Once logged in - it's possible to list the Subscriptions associated with the account via:

```shell
$ az account list
```

The output (similar to below) will display one or more Subscriptions - with the `ID` field being the `subscription_id` field referenced above.

```json
[
  {
    "cloudName": "AzureCloud",
    "id": "00000000-0000-0000-0000-000000000000",
    "isDefault": true,
    "name": "PAYG Subscription",
    "state": "Enabled",
    "tenantId": "00000000-0000-0000-0000-000000000000",
    "user": {
      "name": "user@example.com",
      "type": "user"
    }
  }
]
```

Should you have more than one Subscription, you can specify the Subscription to use via the following command:

```shell
$ az account set --subscription="SUBSCRIPTION_ID"
```

We can now create the Service Principal, which will have permissions to manage resources in the specified Subscription using the following command:

```shell
$ az ad sp create-for-rbac --role="Contributor" --scopes="/subscriptions/SUBSCRIPTION_ID"
```

This command will output 5 values:

```json
{
  "appId": "00000000-0000-0000-0000-000000000000",
  "displayName": "azure-cli-2017-06-05-10-41-15",
  "name": "http://azure-cli-2017-06-05-10-41-15",
  "password": "0000-0000-0000-0000-000000000000",
  "tenant": "00000000-0000-0000-0000-000000000000"
}
```

These values map to the Terraform variables like so:

 - `appId` is the `client_id` defined above.
 - `password` is the `client_secret` defined above.
 - `tenant` is the `tenant_id` defined above.

---

Finally - it's possible to test these values work as expected by first logging in:

```shell
$ az login --service-principal -u CLIENT_ID -p CLIENT_SECRET --tenant TENANT_ID
```

Once logged in as the Service Principal - we should be able to list the VM Sizes by specifying an Azure region, for example here we use the `West US` region:

```shell
$ az vm list-sizes --location westus
```

~> **Note**: If you're using the **China**, **German** or **Government** Azure Clouds - you will need to switch `westus` out for another region. You can find which regions are available by running:

```
$ az account list-locations
```

### Creating Credentials in the Azure Portal

There's a couple of phases to create Credentials via [the Azure Portal](https://portal.azure.com):

 1. Creating an Application in Azure Active Directory (which acts as a Service Principal)
 2. Granting the Application access to manage resources in your Azure Subscription

### 1. Creating an Application in Azure Active Directory

Firstly navigate to [the **Azure Active Directory** overview](https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/Overview) within the Azure Portal - [then select the **App Registration** blade](https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/RegisteredApps/RegisteredApps/Overview) and finally click **Endpoints** at the top of the **App Registration** blade. This will display a list of URIs, the URI for **OAUTH 2.0 AUTHORIZATION ENDPOINT** contains a GUID - which is your Tenant ID / the `tenant_id` field mentioned above.

Next, navigate back to [the **App Registration** blade](https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/RegisteredApps/RegisteredApps/Overview) - from here we'll create the Application in Azure Active Directory. To do this click **Add** at the top to add a new Application within Azure Active Directory. On this page, set the following values then press **Create**:

- **Name** - this is a friendly identifier and can be anything (e.g. "Terraform")
- **Application Type** - this should be set to "Web app / API"
- **Sign-on URL** - this can be anything, providing it's a valid URI (e.g. https://terra.form)

Once that's done - select the Application you just created in [the **App Registration** blade](https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/RegisteredApps/RegisteredApps/Overview). At the top of this page, the "Application ID" GUID is the `client_id` you'll need.

Finally, we can create the `client_secret` by selecting **Keys** and then generating a new key by entering a description, selecting how long the `client_secret` should be valid for - and finally pressing **Save**. This value will only be visible whilst on the page, so be sure to copy it now (otherwise you'll need to regenerate a new key).

### 2. Granting the Application access to manage resources in your Azure Subscription

Once the Application exists in Azure Active Directory - we can grant it permissions to modify resources in the Subscription. To do this, [navigate to the **Subscriptions** blade within the Azure Portal](https://portal.azure.com/#blade/Microsoft_Azure_Billing/SubscriptionsBlade), then select the Subscription you wish to use, then click **Access Control (IAM)**, and finally **Add**.

Firstly specify a Role which grants the appropriate permissions needed for the Service Principal (for example, `Contributor` will grant Read/Write on all resources in the Subscription). There's more information about [the built in roles](https://azure.microsoft.com/en-gb/documentation/articles/role-based-access-built-in-roles/) available here.

Secondly, search for and select the name of the Application created in Azure Active Directory to assign it this role - then press **Save**.

## Creating Credentials through the Legacy CLI's

It's also possible to create credentials via [the legacy cross-platform CLI](https://azure.microsoft.com/en-us/documentation/articles/resource-group-authenticate-service-principal-cli/) and the [legacy PowerShell Commandlets](https://azure.microsoft.com/en-us/documentation/articles/resource-group-authenticate-service-principal/) - however we would highly recommend using the Azure CLI above.

## Testing

Credentials must be provided via the `ARM_SUBSCRIPTION_ID`, `ARM_CLIENT_ID`,
`ARM_CLIENT_SECRET` and `ARM_TENANT_ID` environment variables in order to run
acceptance tests.
