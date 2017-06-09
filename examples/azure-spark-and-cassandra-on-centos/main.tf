# provider "azurerm" {
#   subscription_id = "${var.subscription_id}"
#   client_id       = "${var.client_id}"
#   client_secret   = "${var.client_secret}"
#   tenant_id       = "${var.tenant_id}"
# }

resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group}"
  location = "${var.location}"
}

# **********************  NETWORK SECURITY GROUPS ********************** #
resource "azurerm_network_security_group" "master" {
  name                = "${var.nsg_spark_master_name}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"

  security_rule {
    name                       = "ssh"
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
    name                       = "http_webui_spark"
    description                = "Allow Web UI Access to Spark"
    priority                   = 101
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "8080"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "http_rest_spark"
    description                = "Allow REST API Access to Spark"
    priority                   = 102
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "6066"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }
}

resource "azurerm_network_security_group" "slave" {
  name                = "${var.nsg_spark_slave_name}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"

  security_rule {
    name                       = "ssh"
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
}

resource "azurerm_network_security_group" "cassandra" {
  name                = "${var.nsg_cassandra_name}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"

  security_rule {
    name                       = "ssh"
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
}

# **********************  VNET / SUBNETS ********************** #
resource "azurerm_virtual_network" "spark" {
  name                = "vnet-spark"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  address_space       = ["${var.vnet_spark_prefix}"]
}

resource "azurerm_subnet" "subnet1" {
  name                      = "${var.vnet_spark_subnet1_name}"
  virtual_network_name      = "${azurerm_virtual_network.spark.name}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  address_prefix            = "${var.vnet_spark_subnet1_prefix}"
  network_security_group_id = "${azurerm_network_security_group.master.id}"
  depends_on                = ["azurerm_virtual_network.spark"]
}

resource "azurerm_subnet" "subnet2" {
  name                 = "${var.vnet_spark_subnet2_name}"
  virtual_network_name = "${azurerm_virtual_network.spark.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "${var.vnet_spark_subnet2_prefix}"
}

resource "azurerm_subnet" "subnet3" {
  name                 = "${var.vnet_spark_subnet3_name}"
  virtual_network_name = "${azurerm_virtual_network.spark.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "${var.vnet_spark_subnet3_prefix}"
}

# **********************  PUBLIC IP ADDRESSES ********************** #
resource "azurerm_public_ip" "master" {
  name                         = "${var.public_ip_master_name}"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Static"
}

resource "azurerm_public_ip" "slave" {
  name                         = "${var.public_ip_slave_name_prefix}${count.index}"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Static"
  count                        = "${var.vm_number_of_slaves}"
}

resource "azurerm_public_ip" "cassandra" {
  name                         = "${var.public_ip_cassandra_name}"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Static"
}

# **********************  NETWORK INTERFACE ********************** #
resource "azurerm_network_interface" "master" {
  name                      = "${var.nic_master_name}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.master.id}"
  depends_on                = ["azurerm_virtual_network.spark", "azurerm_public_ip.master", "azurerm_network_security_group.master"]

  ip_configuration {
    name                          = "ipconfig1"
    subnet_id                     = "${azurerm_subnet.subnet1.id}"
    private_ip_address_allocation = "Static"
    private_ip_address            = "${var.nic_master_node_ip}"
    public_ip_address_id          = "${azurerm_public_ip.master.id}"
  }
}

resource "azurerm_network_interface" "slave" {
  name                      = "${var.nic_slave_name_prefix}${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.slave.id}"
  count                     = "${var.vm_number_of_slaves}"
  depends_on                = ["azurerm_virtual_network.spark", "azurerm_public_ip.slave", "azurerm_network_security_group.slave"]

  ip_configuration {
    name                          = "ipconfig1"
    subnet_id                     = "${azurerm_subnet.subnet2.id}"
    private_ip_address_allocation = "Static"
    private_ip_address            = "${var.nic_slave_node_ip_prefix}${5 + count.index}"
    public_ip_address_id          = "${element(azurerm_public_ip.slave.*.id, count.index)}"
  }
}

resource "azurerm_network_interface" "cassandra" {
  name                = "${var.nic_cassandra_name}"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  network_security_group_id     = "${azurerm_network_security_group.cassandra.id}"
  depends_on          = ["azurerm_virtual_network.spark", "azurerm_public_ip.cassandra", "azurerm_network_security_group.cassandra"]

  ip_configuration {
    name                          = "ipconfig1"
    subnet_id                     = "${azurerm_subnet.subnet3.id}"
    private_ip_address_allocation = "Static"
    private_ip_address            = "${var.nic_cassandra_node_ip}"
    public_ip_address_id          = "${azurerm_public_ip.cassandra.id}"
  }
}

# **********************  AVAILABILITY SET ********************** #
resource "azurerm_availability_set" "slave" {
  name                         = "${var.availability_slave_name}"
  location                     = "${azurerm_resource_group.rg.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  platform_update_domain_count = 5
  platform_fault_domain_count  = 2
}

# **********************  STORAGE ACCOUNTS ********************** #
resource "azurerm_storage_account" "master" {
  name                = "master${var.unique_prefix}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_master_type}"
}

resource "azurerm_storage_container" "master" {
  name                  = "${var.vm_master_storage_account_container_name}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  storage_account_name  = "${azurerm_storage_account.master.name}"
  container_access_type = "private"
  depends_on            = ["azurerm_storage_account.master"]
}

resource "azurerm_storage_account" "slave" {
  name                = "slave${var.unique_prefix}${count.index}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  count               = "${var.vm_number_of_slaves}"
  account_type        = "${var.storage_slave_type}"
}

resource "azurerm_storage_container" "slave" {
  name                  = "${var.vm_slave_storage_account_container_name}${count.index}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  storage_account_name  = "${element(azurerm_storage_account.slave.*.name, count.index)}"
  container_access_type = "private"
  depends_on          = ["azurerm_storage_account.slave"]
}

resource "azurerm_storage_account" "cassandra" {
  name                = "cassandra${var.unique_prefix}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_cassandra_type}"
}

resource "azurerm_storage_container" "cassandra" {
  name                  = "${var.vm_cassandra_storage_account_container_name}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  storage_account_name  = "${azurerm_storage_account.cassandra.name}"
  container_access_type = "private"
  depends_on          = ["azurerm_storage_account.cassandra"]
}

# ********************** MASTER VIRTUAL MACHINE ********************** #
resource "azurerm_virtual_machine" "master" {
  name                  = "${var.vm_master_name}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  location              = "${azurerm_resource_group.rg.location}"
  vm_size               = "${var.vm_master_vm_size}"
  network_interface_ids = ["${azurerm_network_interface.master.id}"]
  depends_on            = ["azurerm_storage_account.master", "azurerm_network_interface.master", "azurerm_storage_container.master"]

  storage_image_reference {
    publisher = "${var.os_image_publisher}"
    offer     = "${var.os_image_offer}"
    sku       = "${var.os_version}"
    version   = "latest"
  }

  storage_os_disk {
    name          = "${var.vm_master_os_disk_name}"
    vhd_uri       = "http://${azurerm_storage_account.master.name}.blob.core.windows.net/${azurerm_storage_container.master.name}/${var.vm_master_os_disk_name}.vhd"
    create_option = "FromImage"
    caching       = "ReadWrite"
  }

  os_profile {
    computer_name  = "${var.vm_master_name}"
    admin_username = "${var.vm_admin_username}"
    admin_password = "${var.vm_admin_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }

  connection {
    type     = "ssh"
    host     = "${azurerm_public_ip.master.ip_address}"
    user     = "${var.vm_admin_username}"
    password = "${var.vm_admin_password}"
  }

  provisioner "remote-exec" {
    inline = [
      "wget ${var.artifacts_location}${var.script_spark_provisioner_script_file_name}",
      "echo ${var.vm_admin_password} | sudo -S sh ./${var.script_spark_provisioner_script_file_name} -runas=master -master=${var.nic_master_node_ip}",
    ]
  }
}

# ********************** SLAVE VIRTUAL MACHINES ********************** #
resource "azurerm_virtual_machine" "slave" {
  name                  = "${var.vm_slave_name_prefix}${count.index}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  location              = "${azurerm_resource_group.rg.location}"
  vm_size               = "${var.vm_slave_vm_size}"
  network_interface_ids = ["${element(azurerm_network_interface.slave.*.id, count.index)}"]
  count                 = "${var.vm_number_of_slaves}"
  availability_set_id   = "${azurerm_availability_set.slave.id}"
  depends_on            = ["azurerm_storage_account.slave", "azurerm_network_interface.slave", "azurerm_storage_container.slave"]


  storage_image_reference {
    publisher = "${var.os_image_publisher}"
    offer     = "${var.os_image_offer}"
    sku       = "${var.os_version}"
    version   = "latest"
  }


  storage_os_disk {
    name          = "${var.vm_slave_os_disk_name_prefix}${count.index}"
    vhd_uri       = "http://${element(azurerm_storage_account.slave.*.name, count.index)}.blob.core.windows.net/${element(azurerm_storage_container.slave.*.name, count.index)}/${var.vm_slave_os_disk_name_prefix}.vhd"
    create_option = "FromImage"
    caching       = "ReadWrite"
  }


  os_profile {
    computer_name  = "${var.vm_slave_name_prefix}${count.index}"
    admin_username = "${var.vm_admin_username}"
    admin_password = "${var.vm_admin_password}"
  }


  os_profile_linux_config {
    disable_password_authentication = false
  }
  
  connection {
    type     = "ssh"
    host     = "${element(azurerm_public_ip.slave.*.ip_address, count.index)}"
    user     = "${var.vm_admin_username}"
    password = "${var.vm_admin_password}"
  }

  provisioner "remote-exec" {
    inline = [
      "wget ${var.artifacts_location}${var.script_spark_provisioner_script_file_name}",
      "echo ${var.vm_admin_password} | sudo -S sh ./${var.script_spark_provisioner_script_file_name} -runas=slave -master=${var.nic_master_node_ip}",
    ]
  }
}

# ********************** CASSANDRA VIRTUAL MACHINE ********************** #
resource "azurerm_virtual_machine" "cassandra" {
  name                  = "${var.vm_cassandra_name}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  location              = "${azurerm_resource_group.rg.location}"
  vm_size               = "${var.vm_cassandra_vm_size}"
  network_interface_ids = ["${azurerm_network_interface.cassandra.id}"]
  depends_on            = ["azurerm_storage_account.cassandra", "azurerm_network_interface.cassandra", "azurerm_storage_container.cassandra"]

  storage_image_reference {
    publisher = "${var.os_image_publisher}"
    offer     = "${var.os_image_offer}"
    sku       = "${var.os_version}"
    version   = "latest"
  }

  storage_os_disk {
    name          = "${var.vm_cassandra_os_disk_name}"
    vhd_uri       = "http://${azurerm_storage_account.cassandra.name}.blob.core.windows.net/${azurerm_storage_container.cassandra.name}/${var.vm_cassandra_os_disk_name}.vhd"
    create_option = "FromImage"
    caching       = "ReadWrite"
  }

  os_profile {
    computer_name  = "${var.vm_cassandra_name}"
    admin_username = "${var.vm_admin_username}"
    admin_password = "${var.vm_admin_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }

  connection {
    type     = "ssh"
    host     = "${azurerm_public_ip.cassandra.ip_address}"
    user     = "${var.vm_admin_username}"
    password = "${var.vm_admin_password}"
  }

  provisioner "remote-exec" {
    inline = [
      "wget ${var.artifacts_location}${var.script_cassandra_provisioner_script_file_name}",
      "echo ${var.vm_admin_password} | sudo -S sh ./${var.script_cassandra_provisioner_script_file_name}",
    ]
  }
}
