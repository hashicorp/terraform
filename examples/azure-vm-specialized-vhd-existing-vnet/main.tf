resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}

resource "azurerm_virtual_network" "vnet" {
  name                = "${var.existing_virtual_network_name}"
  location            = "${var.location}"
  address_space       = ["10.0.0.0/16"]
  resource_group_name = "${azurerm_resource_group.rg.name}"
}

resource "azurerm_subnet" "subnet1" {
  name                 = "subnet1"
  virtual_network_name = "${azurerm_virtual_network.vnet.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "10.0.0.0/24"
}

resource "azurerm_network_interface" "nic" {
  name                = "nic"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  ip_configuration {
    name                          = "ipconfig"
    subnet_id                     = "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = "${azurerm_public_ip.pip.id}"
  }
}

resource "azurerm_public_ip" "pip" {
  name                         = "PublicIp"
  location                     = "${var.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "dynamic"
  domain_name_label            = "${var.dns_name}"
}

resource "azurerm_storage_account" "stor" {
  name                = "diagStorageAcct"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${var.location}"
  account_type        = "Standard_GRS"
}

resource "azurerm_virtual_machine" "vm" {
  name                  = "${var.hostname}"
  location              = "${var.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${azurerm_network_interface.nic.id}"]

  storage_os_disk {
    name          = "${var.dns_name}-osdisk1"
    vhd_uri       = "https://permanentstor.blob.core.windows.net/permanent-vhds/permanent-osdisk1.vhd" # "https://${var.storage_account_name}.blob.core.windows.net/vhds/${var.hostname}osdisk.vhd"
    os_type       = "${var.os_type}"
    caching       = "ReadWrite"
    create_option = "Attach"
  }

  os_profile {
    computer_name  = "${var.hostname}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.admin_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }

  boot_diagnostics {
    enabled     = "true"
    storage_uri = "${azurerm_storage_account.stor.primary_blob_endpoint}"
  }
}
