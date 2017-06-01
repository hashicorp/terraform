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

resource "azurerm_virtual_network" "vnet" {
  name                = "${var.hostname}vnet"
  location            = "${var.location}"
  address_space       = ["${var.address_space}"]
  resource_group_name = "${azurerm_resource_group.rg.name}"
}

resource "azurerm_subnet" "subnet" {
  name                 = "${var.hostname}subnet"
  virtual_network_name = "${azurerm_virtual_network.vnet.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "${var.subnet_prefix}"
}

resource "azurerm_network_interface" "nic" {
  name                = "nic"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  ip_configuration {
    name                          = "ipconfig"
    subnet_id                     = "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_storage_account" "stor" {
  name                = "${var.hostname}stor"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_virtual_machine" "vm" {
  name                  = "${var.hostname}"
  location              = "${var.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${azurerm_network_interface.nic.id}"]

  storage_image_reference {
    publisher = "${var.image_publisher}"
    offer     = "${var.image_offer}"
    sku       = "${var.image_sku}"
    version   = "${var.image_version}"
  }

  storage_os_disk {
    name          = "${var.hostname}osdisk"
    create_option = "FromImage"
    disk_size_gb  = "15"
  }

  os_profile {
    computer_name  = "${var.hostname}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.admin_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }
}

resource "azurerm_template_deployment" "linux_vm" {
  name                = "encrypt"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  deployment_mode     = "Incremental"
  depends_on          = ["azurerm_virtual_machine.vm"]

  template_body = <<DEPLOY
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "aadClientID": {
      "defaultValue": "${var.aad_client_id}",
      "type": "string"
    },
    "aadClientSecret": {
      "defaultValue": "${var.aad_client_secret}",
      "type": "string"
    },
    "diskFormatQuery": {
      "defaultValue": "",
      "type": "string"
    },
    "encryptionOperation": {
      "allowedValues": [ "EnableEncryption", "EnableEncryptionFormat" ],
      "defaultValue": "${var.encryption_operation}",
      "type": "string"
    },
    "volumeType": {
      "allowedValues": [ "OS", "Data", "All" ],
      "defaultValue": "${var.volume_type}",
      "type": "string"
    },
    "keyEncryptionKeyURL": {
      "defaultValue": "${var.key_encryption_key_url}",
      "type": "string"
    },
    "keyVaultName": {
      "defaultValue": "${var.key_vault_name}",
      "type": "string"
    },
    "keyVaultResourceGroup": {
      "defaultValue": "${azurerm_resource_group.rg.name}",
      "type": "string"
    },
    "passphrase": {
      "defaultValue": "${var.passphrase}",
      "type": "string"
    },
    "sequenceVersion": {
      "defaultValue": "${var.sequence_version}",
      "type": "string"
    },
    "useKek": {
      "allowedValues": [
        "nokek",
        "kek"
      ],
      "defaultValue": "${var.use_kek}",
      "type": "string"
    },
    "vmName": {
      "defaultValue": "${azurerm_virtual_machine.vm.name}",
      "type": "string"
    },
    "_artifactsLocation": {
      "type": "string",
      "defaultValue": "${var.artifacts_location}"
    },
    "_artifactsLocationSasToken": {
      "type": "string",
      "defaultValue": "${var.artifacts_location_sas_token}"
    }    
  },
  "variables": {
    "extensionName": "${var.extension_name}",
    "extensionVersion": "0.1",
    "keyEncryptionAlgorithm": "RSA-OAEP",
    "keyVaultURL": "https://${var.key_vault_name}.vault.azure.net/",
    "keyVaultResourceID": "${var.key_vault_resource_id}",
    "updateVmUrl": "${var.artifacts_location}/201-encrypt-running-linux-vm/updatevm-${var.use_kek}.json${var.artifacts_location_sas_token}"
  },
  "resources": [
    {
      "type": "Microsoft.Compute/virtualMachines/extensions",
      "name": "[concat(parameters('vmName'),'/', variables('extensionName'))]",
      "apiVersion": "2015-06-15",
      "location": "[resourceGroup().location]",
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
        }
      }
    }
  ],
  "outputs": {
    "BitLockerKey": {
      "type": "string",
      "value": "[reference(resourceId('Microsoft.Compute/virtualMachines/extensions',  parameters('vmName'), variables('extensionName'))).instanceView.statuses[0].message]"
    }
  }
}
DEPLOY
}
