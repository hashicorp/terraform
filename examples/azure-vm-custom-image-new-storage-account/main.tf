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
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  address_space       = ["${var.address_space}"]
}

resource "azurerm_subnet" "subnet" {
  name                 = "${var.hostname}subnet"
  virtual_network_name = "${azurerm_virtual_network.vnet.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "${var.subnet_prefix}"
}

resource "azurerm_public_ip" "pip" {
  name                         = "PublicIp${count.index}"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Dynamic"
  count                        = "${var.vm_count}"
}

resource "azurerm_network_interface" "nic" {
  name                = "nic${count.index}"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  count               = "${var.vm_count}"

  ip_configuration {
    name                          = "${element(azurerm_public_ip.pip.*.name, count.index)}"
    subnet_id                     = "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = "${element(azurerm_public_ip.pip.*.id, count.index)}"
  }
}

resource "azurerm_storage_account" "existing" {
  name                = "${var.existing_storage_acct}"
  resource_group_name = "${var.existing_resource_group}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.existing_storage_acct_type}"

  lifecycle = {
    prevent_destroy = true
  }
}

resource "azurerm_storage_account" "stor" {
  name                = "${var.hostname}stor"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_virtual_machine" "transfer" {
  name                  = "${var.transfer_vm_name}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${element(azurerm_network_interface.nic.*.id, count.index)}"]

  storage_os_disk {
    name          = "${var.hostname}-osdisk"
    image_uri     = "${var.source_img_uri}"
    vhd_uri       = "https://${var.existing_storage_acct}.blob.core.windows.net/${var.existing_resource_group}-vhds/${var.hostname}3osdisk.vhd"
    os_type       = "${var.os_type}"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "${var.hostname}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.admin_password}"
  }

  boot_diagnostics {
    enabled     = true
    storage_uri = "${azurerm_storage_account.stor.primary_blob_endpoint}"
  }

  provisioner "file" {
    source = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/201-vm-custom-image-new-storage-account/ImageTransfer.ps1"
    destination = "C:/tmp/ImageTransfer.ps1"
       
    connection {
      type     = "winrm"
      user     = "${var.admin_username}"
      password = "${var.admin_password}"
      host     = "${azurerm_public_ip.pip.ip_address}"
    }
  }
  
  provisioner "remote-exec" {
    inline = [
      "powershell -ExecutionPolicy Unrestricted -File ImageTransfer.ps1 -SourceImage ${var.source_img_uri} -SourceSAKey ${azurerm_storage_account.existing.primary_access_key} -DestinationURI https://${azurerm_storage_account.stor.name}.blob.core.windows.net/vhds -DestinationSAKey ${azurerm_storage_account.stor.primary_access_key}"
    ]
   
    connection {
      type     = "winrm"
      user     = "${var.admin_username}"
      password = "${var.admin_password}"
      host     = "${azurerm_public_ip.pip.ip_address}"
    }
  }

}

# ***************************************************************************************************************
# FROM ARM TEMPLATE:
# "commandToExecute": "[concat('powershell -ExecutionPolicy Unrestricted -File ','ImageTransfer.ps1 -SourceImage ',parameters('sourceImageURI'),' -SourceSAKey ', listKeys(resourceId(parameters('sourceStorageAccountResourceGroup'),'Microsoft.Storage/storageAccounts', variables('sourceStorageAccountName')), '2015-06-15').key1, ' -DestinationURI https://', variables('StorageAccountName'), '.blob.core.windows.net/vhds', ' -DestinationSAKey ', listKeys(concat('Microsoft.Storage/storageAccounts/', variables('StorageAccountName')), '2015-06-15').key1)]"
# ***************************************************************************************************************
