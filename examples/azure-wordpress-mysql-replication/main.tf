# provider "azurerm" {
#   subscription_id = "${var.subscription_id}"
#   client_id       = "${var.client_id}"
#   client_secret   = "${var.client_secret}"
#   tenant_id       = "${var.tenant_id}"
# }

# ********************** MYSQL REPLICATION ********************** #

resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}

# ********************** VNET / SUBNET ********************** #
resource "azurerm_virtual_network" "vnet" {
  name                = "${var.virtual_network_name}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  address_space       = ["${var.vnet_address_prefix}"]
}

resource "azurerm_subnet" "db_subnet" {
  name                      = "${var.db_subnet_name}"
  virtual_network_name      = "${azurerm_virtual_network.vnet.name}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.nsg.id}"
  address_prefix            = "${var.db_subnet_address_prefix}"
  depends_on                = ["azurerm_virtual_network.vnet"]
}

# **********************  STORAGE ACCOUNTS ********************** #
resource "azurerm_storage_account" "stor" {
  name                = "${var.unique_prefix}${var.storage_account_name}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type}"
}

# **********************  NETWORK SECURITY GROUP ********************** #
resource "azurerm_network_security_group" "nsg" {
  name                = "${var.unique_prefix}-nsg"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"

  security_rule {
    name                       = "allow-ssh"
    description                = "Allow SSH"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }

 security_rule {
    name                       = "MySQL"
    description                = "MySQL"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "3306"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

# **********************  PUBLIC IP ADDRESSES ********************** #
resource "azurerm_public_ip" "pip" {
  name                         = "${var.public_ip_name}"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Static"
  domain_name_label            = "${var.dns_name}"
}

# **********************  AVAILABILITY SET ********************** #
resource "azurerm_availability_set" "availability_set" {
  name                = "${var.dns_name}-set"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
}

# **********************  NETWORK INTERFACES ********************** #
resource "azurerm_network_interface" "nic" {
  name                      = "${var.nic_name}${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.nsg.id}"
  count                     = "${var.node_count}"
  depends_on                = ["azurerm_virtual_network.vnet", "azurerm_public_ip.pip", "azurerm_lb.lb"]

  ip_configuration {
    name                                    = "ipconfig${count.index}"
    subnet_id                               = "${azurerm_subnet.db_subnet.id}"
    private_ip_address_allocation           = "Static"
    private_ip_address                      = "10.0.1.${count.index + 4}"
    load_balancer_backend_address_pools_ids = ["${azurerm_lb_backend_address_pool.backend_pool.id}"]

    load_balancer_inbound_nat_rules_ids = [
      "${element(azurerm_lb_nat_rule.NatRule0.*.id, count.index)}",
      "${element(azurerm_lb_nat_rule.MySQLNatRule0.*.id, count.index)}",
      "${element(azurerm_lb_nat_rule.ProbeNatRule0.*.id, count.index)}",
    ]
  }
}

# **********************  LOAD BALANCER ********************** #
resource "azurerm_lb" "lb" {
  name                = "${var.dns_name}-lb"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  depends_on          = ["azurerm_public_ip.pip"]

  frontend_ip_configuration {
    name                 = "${var.dns_name}-sshIPCfg"
    public_ip_address_id = "${azurerm_public_ip.pip.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "backend_pool" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  loadbalancer_id     = "${azurerm_lb.lb.id}"
  name                = "${var.dns_name}-ilbBackendPool"
}

# **********************  LOAD BALANCER INBOUND NAT RULES ********************** #
resource "azurerm_lb_nat_rule" "NatRule0" {
  name                           = "${var.dns_name}-NatRule-${count.index}"
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.lb.id}"
  protocol                       = "tcp"
  frontend_port                  = "6400${count.index + 1}"
  backend_port                   = 22
  frontend_ip_configuration_name = "${var.dns_name}-sshIPCfg"
  count                          = "${var.node_count}"
  depends_on                     = ["azurerm_lb.lb"]
}

resource "azurerm_lb_nat_rule" "MySQLNatRule0" {
  name                           = "${var.dns_name}-MySQLNatRule-${count.index}"
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.lb.id}"
  protocol                       = "tcp"
  frontend_port                  = "330${count.index + 6}"
  backend_port                   = 3306
  frontend_ip_configuration_name = "${var.dns_name}-sshIPCfg"
  count                          = "${var.node_count}"
  depends_on                     = ["azurerm_lb.lb"]
}

resource "azurerm_lb_nat_rule" "ProbeNatRule0" {
  name                           = "${var.dns_name}-ProbeNatRule-${count.index}"
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.lb.id}"
  protocol                       = "tcp"
  frontend_port                  = "920${count.index}"
  backend_port                   = 9200
  frontend_ip_configuration_name = "${var.dns_name}-sshIPCfg"
  count                          = "${var.node_count}"
  depends_on                     = ["azurerm_lb.lb"]
}

# ********************** VIRTUAL MACHINES ********************** #
resource "azurerm_virtual_machine" "vm" {
  name                  = "${var.dns_name}${count.index}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  location              = "${azurerm_resource_group.rg.location}"
  vm_size               = "${var.vm_size}"
  network_interface_ids = ["${element(azurerm_network_interface.nic.*.id, count.index)}"]
  count                 = "${var.node_count}"
  availability_set_id   = "${azurerm_availability_set.availability_set.id}"
  depends_on            = ["azurerm_availability_set.availability_set", "azurerm_network_interface.nic", "azurerm_storage_account.stor"]

  storage_image_reference {
    publisher = "${var.image_publisher}"
    offer     = "${var.image_offer}"
    sku       = "${var.os_version}"
    version   = "latest"
  }

  storage_os_disk {
    name          = "osdisk${count.index}"
    vhd_uri       = "https://${azurerm_storage_account.stor.name}.blob.core.windows.net/vhds/${var.dns_name}${count.index}-osdisk.vhd"
    create_option = "FromImage"
    caching       = "ReadWrite"
  }

  os_profile {
    computer_name  = "${var.dns_name}${count.index}"
    admin_username = "${var.vm_admin_username}"
    admin_password = "${var.vm_admin_password}"
  }

  storage_data_disk {
    name          = "datadisk1"
    vhd_uri       = "https://${azurerm_storage_account.stor.name}.blob.core.windows.net/vhds/${var.dns_name}${count.index}-datadisk1.vhd"
    disk_size_gb  = "1000"
    create_option = "Empty"
    lun           = 0
  }

  storage_data_disk {
    name          = "datadisk2"
    vhd_uri       = "https://${azurerm_storage_account.stor.name}.blob.core.windows.net/vhds/${var.dns_name}${count.index}-datadisk2.vhd"
    disk_size_gb  = "1000"
    create_option = "Empty"
    lun           = 1
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }
}

resource "azurerm_virtual_machine_extension" "setup_mysql" {
  name                       = "${var.dns_name}-${count.index}-setupMySQL"
  resource_group_name        = "${azurerm_resource_group.rg.name}"
  location                   = "${azurerm_resource_group.rg.location}"
  virtual_machine_name       = "${element(azurerm_virtual_machine.vm.*.name, count.index)}"
  publisher                  = "Microsoft.Azure.Extensions"
  type                       = "CustomScript"
  type_handler_version       = "2.0"
  auto_upgrade_minor_version = true
  count                      = "${var.node_count}"
  depends_on                 = ["azurerm_virtual_machine.vm", "azurerm_lb_nat_rule.ProbeNatRule0"]

  settings = <<SETTINGS
{
  "fileUris": ["${var.artifacts_location}${var.azuremysql_script}"]
}
SETTINGS

  protected_settings = <<SETTINGS
 {
   "commandToExecute": "bash azuremysql.sh ${count.index + 1} 10.0.1.${count.index + 4} ${var.artifacts_location}${var.mysql_cfg_file_path} '${var.mysql_replication_password}' '${var.mysql_root_password}' '${var.mysql_probe_password}' 10.0.1.4 ${var.unique_prefix}wordpress"
 }
SETTINGS
}
