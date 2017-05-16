---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_template_deployment"
sidebar_current: "docs-azurerm-resource-template-deployment"
description: |-
  Create a template deployment of resources.
---

# azurerm\_template\_deployment

Create a template deployment of resources

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acctestrg-01"
  location = "West US"
}

resource "azurerm_template_deployment" "test" {
  name                = "acctesttemplate-01"
  resource_group_name = "${azurerm_resource_group.test.name}"

  template_body = <<DEPLOY
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "storageAccountType": {
      "type": "string",
      "defaultValue": "Standard_LRS",
      "allowedValues": [
        "Standard_LRS",
        "Standard_GRS",
        "Standard_ZRS"
      ],
      "metadata": {
        "description": "Storage Account type"
      }
    }
  },
  "variables": {
    "location": "[resourceGroup().location]",
    "storageAccountName": "[concat(uniquestring(resourceGroup().id), 'storage')]",
    "publicIPAddressName": "[concat('myPublicIp', uniquestring(resourceGroup().id))]",
    "publicIPAddressType": "Dynamic",
    "apiVersion": "2015-06-15",
    "dnsLabelPrefix": "terraform-acctest"
  },
  "resources": [
    {
      "type": "Microsoft.Storage/storageAccounts",
      "name": "[variables('storageAccountName')]",
      "apiVersion": "[variables('apiVersion')]",
      "location": "[variables('location')]",
      "properties": {
        "accountType": "[parameters('storageAccountType')]"
      }
    },
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "[variables('apiVersion')]",
      "name": "[variables('publicIPAddressName')]",
      "location": "[variables('location')]",
      "properties": {
        "publicIPAllocationMethod": "[variables('publicIPAddressType')]",
        "dnsSettings": {
          "domainNameLabel": "[variables('dnsLabelPrefix')]"
        }
      }
    }
  ],
  "outputs": {
    "storageAccountName": {
      "type": "string",
      "value": "[variables('storageAccountName')]"
    }
  }
}
DEPLOY

  deployment_mode = "Incremental"
}

output "storageAccountName" {
  value = "${azurerm_template_deployment.test.outputs["storageAccountName"]}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the template deployment. Changing this forces a
    new resource to be created.
* `resource_group_name` - (Required) The name of the resource group in which to
    create the template deployment.
* `deployment_mode` - (Required) Specifies the mode that is used to deploy resources. This value could be either `Incremental` or `Complete`.
    Note that you will almost *always* want this to be set to `Incremental` otherwise the deployment will destroy all infrastructure not
    specified within the template, and Terraform will not be aware of this.
* `template_body` - (Optional) Specifies the JSON definition for the template.
* `parameters` - (Optional) Specifies the name and value pairs that define the deployment parameters for the template.

## Attributes Reference

The following attributes are exported:

* `id` - The Template Deployment ID.

* `outputs` - A map of supported scalar output types returned from the deployment (currently, Azure Template Deployment outputs of type String, Int and Bool are supported, and are converted to strings - others will be ignored) and can be accessed using `.outputs["name"]`.

## Note

Terraform does not know about the individual resources created by Azure using a deployment template and therefore cannot delete these resources during a destroy. Destroying a template deployment removes the associated deployment operations, but will not delete the Azure resources created by the deployment. In order to delete these resources, the containing resource group must also be destroyed. [More information](https://docs.microsoft.com/en-us/rest/api/resources/deployments#Deployments_Delete).
