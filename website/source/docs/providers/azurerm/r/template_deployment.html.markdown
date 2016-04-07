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

```
resource "azurerm_resource_group" "test" {
    name = "acctestrg-01"
    location = "West US"
  }

  resource "azurerm_template_deployment" "test" {
    name = "acctesttemplate-01"
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
  ]
}
DEPLOY
    deployment_mode = "Complete"
  }
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the template deployment. Changing this forces a
    new resource to be created.
* `resource_group_name` - (Required) The name of the resource group in which to
    create the template deployment.
* `template_body` - (Optional) Specifies the JSON definition for the template.
* `parameters` - (Optional) Specifies the name and value pairs that define the deployment parameters for the template.
* `deployment_mode` - (Optional) Specifies the mode that is used to deploy resources. This value could be either `Incremental` or `Complete`. 

## Attributes Reference

The following attributes are exported:

* `id` - The Template Deployment ID.