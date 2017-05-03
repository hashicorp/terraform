resource "azurerm_network_interface" "nic" {
  name                = "nic"
  location            = "${var.location}"
  resource_group_name = "${var.existing_vnet_resource_group}"

  ip_configuration {
    name                          = "ipconfig"
    subnet_id                     = "${var.subnet_id}" # "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = "${azurerm_public_ip.pip.id}"
  }
}

resource "azurerm_public_ip" "pip" {
  name                         = "PublicIp"
  location                     = "${var.location}"
  resource_group_name          = "${var.existing_vnet_resource_group}"
  public_ip_address_allocation = "Dynamic"
  domain_name_label            = "${var.hostname}"
}

resource "azurerm_storage_account" "stor" {
  name                = "bootdiagstor"
  resource_group_name = "${var.existing_vnet_resource_group}"
  location            = "${var.location}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_virtual_machine" "vm" {
  name                  = "${var.hostname}"
  location              = "${var.location}"
  resource_group_name   = "${var.existing_vnet_resource_group}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${azurerm_network_interface.nic.id}"]

  storage_os_disk {
    name          = "${var.hostname}osdisk1"
    vhd_uri       = "${var.os_disk_vhd_uri}"  # "https://${var.storage_account_name}.blob.core.windows.net/vhds/${var.hostname}osdisk.vhd"
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
    enabled     = true
    storage_uri = "${azurerm_storage_account.stor.primary_blob_endpoint}"
  }
}
