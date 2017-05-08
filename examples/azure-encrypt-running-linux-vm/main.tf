# provider "azurerm" {
#   subscription_id = "REPLACE-WITH-YOUR-SUBSCRIPTION-ID"
#   client_id       = "REPLACE-WITH-YOUR-CLIENT-ID"
#   client_secret   = "REPLACE-WITH-YOUR-CLIENT-SECRET"
#   tenant_id       = "REPLACE-WITH-YOUR-TENANT-ID"
# }

resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}


resource "azurerm_virtual_machine" "vm" {
  name                  = "vm${count.index}"
  location              = "${var.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.avset.id}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${element(azurerm_network_interface.nic.*.id, count.index)}"]
  count                 = 2

  storage_image_reference {
    publisher = "${var.image_publisher}"
    offer     = "${var.image_offer}"
    sku       = "${var.image_sku}"
    version   = "${var.image_version}"
  }

  storage_os_disk {
    name          = "osdisk${count.index}"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "${var.hostname}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.admin_password}"
  }
}

resource "azurerm_key_vault" "key_vault" {
  name                = "${var.keyvault_name}"
  location            = "${azurerm_resource_group.quickstart.location}"
  resource_group_name = "${azurerm_resource_group.quickstart.name}"

  sku {
    name = "${lookup(var.sku_name_map, var.sku_name)}"
  }

  tenant_id = "${var.keyvault_tenant_id}"

  access_policy {
    tenant_id = "${var.keyvault_tenant_id}"
    object_id = "${var.keyvault_object_id}"

    key_permissions    = "${var.keys_permissions}"
    secret_permissions = "${var.secrets_permissions}"
  }

  enabled_for_deployment          = "${lookup(var.boolean_map, var.enable_vault_for_deployment)}"
  enabled_for_disk_encryption     = "${lookup(var.boolean_map, var.enable_vault_for_disk_encryption)}"
  enabled_for_template_deployment = "${lookup(var.boolean_map, var.enabled_for_template_deployment)}"
}

output "vault_uri" {
  value = ["${azurerm_key_vault.quickstart.vault_uri}"]
}

resource "azurerm_virtual_machine_extension" "ext" {
  name                 = "${var.vm_name}ext"
  location             = "${var.location}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  virtual_machine_name = "${azurerm_virtual_machine.vm.name}"
  publisher            = "Microsoft.OSTCExtensions"
  type                 = "CustomScriptForLinux"
  type_handler_version = "1.2"

  settings = <<SETTINGS
    {
        "commandToExecute": "hostname"
    }
SETTINGS

  tags {
    environment = "Production"
  }
}
  "resources": [

      "properties": {
        "protectedSettings": {
          "AADClientSecret": "[parameters('aadClientSecret')]",
          "Passphrase": "[parameters('passphrase')]"
        },
        "publisher": "Microsoft.Azure.Security",
        "settings": {
          "AADClientID": "[parameters('aadClientID')]",
          "DiskFormatQuery": "[parameters('diskFormatQuery')]",
          "EncryptionOperation": "[parameters('encryptionOperation')]",
          "KeyEncryptionAlgorithm": "[variables('keyEncryptionAlgorithm')]",
          "KeyEncryptionKeyURL": "[parameters('keyEncryptionKeyURL')]",
          "KeyVaultURL": "[variables('keyVaultURL')]",
          "SequenceVersion": "[parameters('sequenceVersion')]",
          "VolumeType": "[parameters('volumeType')]"
        },
        "type": "AzureDiskEncryptionForLinux",
        "typeHandlerVersion": "[variables('extensionVersion')]"
      }
    },
    {
      "apiVersion": "2015-01-01",
      "dependsOn": [
        "[resourceId('Microsoft.Compute/virtualMachines/extensions',  parameters('vmName'), variables('extensionName'))]"
      ],
      "name": "[concat(parameters('vmName'), 'updateVm')]",
      "type": "Microsoft.Resources/deployments",
      "properties": {
        "mode": "Incremental",
        "parameters": {
          "keyEncryptionKeyURL": {
            "value": "[parameters('keyEncryptionKeyURL')]"
          },
          "keyVaultResourceID": {
            "value": "[variables('keyVaultResourceID')]"
          },
          "keyVaultSecretUrl": {
            "value": "[reference(resourceId('Microsoft.Compute/virtualMachines/extensions',  parameters('vmName'), variables('extensionName'))).instanceView.statuses[0].message]"
          },
          "vmName": {
            "value": "[parameters('vmName')]"
          }
        },
        "templateLink": {
          "contentVersion": "1.0.0.0",
          "uri": "[variables('updateVmUrl')]"

            "variables": {
    "extensionName": "AzureDiskEncryptionForLinux",
    "extensionVersion": "0.1",
    "keyEncryptionAlgorithm": "RSA-OAEP",
    "updateVmUrl": "[concat(parameters('_artifactsLocation'), '/', '201-encrypt-running-linux-vm', '/', 'updatevm-', parameters('useKek'), '.json', parameters('_artifactsLocationSasToken'))]",
    "keyVaultURL": "[concat('https://', parameters('keyVaultName'), '.vault.azure.net/')]",
    "keyVaultResourceID": "[concat(subscription().id,'/resourceGroups/',parameters('keyVaultResourceGroup'),'/providers/Microsoft.KeyVault/vaults/', parameters('keyVaultName'))]"
  },


*************************KEY ENCRYPTION KEY***************************************************************************************
{
    "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "vmName": {
            "type": "string",
            "metadata": {
                "description": "Name of the Virtual Machine"
            }
        },
        "keyVaultResourceID": {
            "type": "string",
            "metadata": {
                "description": "KeyVault resource id. Ex: /subscriptions/9135e259-1f76-4dbd-a5c8-bc4fcdf3cf1c/resourceGroups/DiskEncryptionTest/providers/Microsoft.KeyVault/vaults/DiskEncryptionTestAus"
            }
        },
        "keyVaultSecretUrl": {
            "type": "string",
            "metadata": {
                "description": "KeyVault secret Url. Ex: https://diskencryptiontestaus.vault.azure.net/secrets/BitLockerEncryptionSecretWithKek/e088818e865e48488cf363af16dea596"
            }
        },
        "keyEncryptionKeyURL": {
            "type": "string",
            "defaultValue": "",
            "metadata": {
                "description": "KeyVault key encryption key Url. Ex: https://diskencryptiontestaus.vault.azure.net/keys/DiskEncryptionKek/562a4bb76b524a1493a6afe8e536ee78"
            }
        }
    },
    "resources": [
        {
            "apiVersion": "2016-04-30-preview",
            "type": "Microsoft.Compute/virtualMachines",
            "name": "[parameters('vmName')]",
            "location": "[resourceGroup().location]",
            "properties": {
                "storageProfile": {
                    "osDisk": {
                        "encryptionSettings": {
                            "diskEncryptionKey": {
                                "sourceVault": {
                                    "id": "[parameters('keyVaultResourceID')]"
                                },
                                "secretUrl": "[parameters('keyVaultSecretUrl')]"
                            },
                            "keyEncryptionKey": {
                                "sourceVault": {
                                    "id": "[parameters('keyVaultResourceID')]"
                                },
                                "keyUrl": "[parameters('keyEncryptionKeyURL')]"
                            }
                        }
                    }
                }
            }
        }
    ]
******************************NO KEY ENCRYPTION KEY**********************************************************************************

{
    "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "vmName": {
            "type": "string",
            "metadata": {
                "description": "Name of the Virtual Machine"
            }
        },
        "keyVaultResourceID": {
            "type": "string",
            "metadata": {
                "description": "KeyVault resource id. Ex: /subscriptions/9135e259-1f76-4dbd-a5c8-bc4fcdf3cf1c/resourceGroups/DiskEncryptionTest/providers/Microsoft.KeyVault/vaults/DiskEncryptionTestAus"
            }
        },
        "keyVaultSecretUrl": {
            "type": "string",
            "metadata": {
                "description": "KeyVault secret Url. Ex: https://diskencryptiontestaus.vault.azure.net/secrets/BitLockerEncryptionSecretWithKek/e088818e865e48488cf363af16dea596"
            }
        },
        "keyEncryptionKeyURL": {
            "type": "string",
            "defaultValue": "",
            "metadata": {
                "description": "KeyVault key encryption key Url. Ex: https://diskencryptiontestaus.vault.azure.net/keys/DiskEncryptionKek/562a4bb76b524a1493a6afe8e536ee78"
            }
        }
    },
    "resources": [
        {
            "apiVersion": "2016-04-30-preview",
            "type": "Microsoft.Compute/virtualMachines",
            "name": "[parameters('vmName')]",
            "location": "[resourceGroup().location]",
            "properties": {
                "storageProfile": {
                    "osDisk": {
                        "encryptionSettings": {
                            "diskEncryptionKey": {
                                "sourceVault": {
                                    "id": "[parameters('keyVaultResourceID')]"
                                },
                                "secretUrl": "[parameters('keyVaultSecretUrl')]"
                            }
                        }
                    }
                }
            }
        }