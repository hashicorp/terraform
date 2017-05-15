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

resource "azurerm_public_ip" "transferpip" {
  name                         = "transferpip"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Static"
}

resource "azurerm_network_interface" "transfernic" {
  name                = "transfernic"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  ip_configuration {
    name                          = "${azurerm_public_ip.transferpip.name}"
    subnet_id                     = "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Static"
    public_ip_address_id          = "${azurerm_public_ip.transferpip.id}"
    private_ip_address            = "10.0.0.5"
  }
}

resource "azurerm_public_ip" "mypip" {
  name                         = "mypip"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_network_interface" "mynic" {
  name                = "mynic"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  ip_configuration {
    name                          = "${azurerm_public_ip.mypip.name}"
    subnet_id                     = "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = "${azurerm_public_ip.mypip.id}"
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
  name                = "${var.hostname}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_virtual_machine" "transfer" {
  name                  = "${var.transfer_vm_name}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${azurerm_network_interface.transfernic.id}"]

  storage_os_disk {
    name          = "${var.hostname}-osdisk"
    image_uri     = "${var.source_img_uri}"
    vhd_uri       = "https://${var.existing_storage_acct}.blob.core.windows.net/${var.existing_resource_group}-vhds/${var.hostname}osdisk.vhd"
    os_type       = "${var.os_type}"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "${var.hostname}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.admin_password}"
  }
}

resource "azurerm_virtual_machine_extension" "script" {
  name                 = "CustomScriptExtension"
  location             = "${azurerm_resource_group.rg.location}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  virtual_machine_name = "${azurerm_virtual_machine.transfer.name}"
  publisher            = "Microsoft.Compute"
  type                 = "CustomScriptExtension"
  type_handler_version = "1.4"
  depends_on           = ["azurerm_virtual_machine.transfer"]

  settings = <<SETTINGS
    {
        "commandToExecute": "powershell -ExecutionPolicy Unrestricted -Command \"Invoke-WebRequest -Uri https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/201-vm-custom-image-new-storage-account/ImageTransfer.ps1 -OutFile C:/ImageTransfer.ps1\" "
    }
SETTINGS
}

resource "azurerm_virtual_machine_extension" "execute" {
  name                 = "CustomScriptExtension"
  location             = "${azurerm_resource_group.rg.location}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  virtual_machine_name = "${azurerm_virtual_machine.transfer.name}"
  publisher            = "Microsoft.Compute"
  type                 = "CustomScriptExtension"
  type_handler_version = "1.4"
  depends_on           = ["azurerm_virtual_machine_extension.script"]

  settings = <<SETTINGS
    {
        "commandToExecute": "powershell -ExecutionPolicy Unrestricted -File C:\\ImageTransfer.ps1 -SourceImage ${var.source_img_uri} -SourceSAKey ${azurerm_storage_account.existing.primary_access_key} -DestinationURI https://${azurerm_storage_account.stor.name}.blob.core.windows.net/vhds -DestinationSAKey ${azurerm_storage_account.stor.primary_access_key}\" "
    }
SETTINGS
}

resource "azurerm_virtual_machine" "myvm" {
  name                  = "${var.new_vm_name}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${azurerm_network_interface.mynic.id}"]
  depends_on            = ["azurerm_virtual_machine_extension.execute"]

  storage_os_disk {
    name          = "${var.hostname}osdisk"
    image_uri     = "https://${azurerm_storage_account.stor.name}.blob.core.windows.net/vhds/${var.custom_image_name}.vhd"
    vhd_uri       = "https://${var.hostname}.blob.core.windows.net/${var.hostname}-vhds/${var.hostname}osdisk.vhd"
    os_type       = "${var.os_type}"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  os_profile {
    computer_name  = "${var.hostname}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.admin_password}"
  }
}
