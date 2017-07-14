provider "azurerm" {
  subscription_id = "${var.subscription_id}"
  client_id       = "${var.aad_client_id}"
  client_secret   = "${var.aad_client_secret}"
  tenant_id       = "${var.tenant_id}"
}

resource "azurerm_resource_group" "rg" {
  name     = "${var.resource_group_name}"
  location = "${var.resource_group_location}"
}

# ******* NETWORK SECURITY GROUPS ***********

resource "azurerm_network_security_group" "master_nsg" {
  name                = "${var.openshift_cluster_prefix}-master-nsg"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  security_rule {
    name                       = "allow_SSH_in_all"
    description                = "Allow SSH in from all locations"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "allow_HTTPS_all"
    description                = "Allow HTTPS connections from all locations"
    priority                   = 200
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "allow_OpenShift_console_in_all"
    description                = "Allow OpenShift Console connections from all locations"
    priority                   = 300
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "8443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_network_security_group" "infra_nsg" {
  name                = "${var.openshift_cluster_prefix}-infra-nsg"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  security_rule {
    name                       = "allow_SSH_in_all"
    description                = "Allow SSH in from all locations"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "allow_HTTPS_all"
    description                = "Allow HTTPS connections from all locations"
    priority                   = 200
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "allow_HTTP_in_all"
    description                = "Allow HTTP connections from all locations"
    priority                   = 300
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_network_security_group" "node_nsg" {
  name                = "${var.openshift_cluster_prefix}-node-nsg"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  security_rule {
    name                       = "allow_SSH_in_all"
    description                = "Allow SSH in from all locations"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "allow_HTTPS_all"
    description                = "Allow HTTPS connections from all locations"
    priority                   = 200
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "allow_HTTP_in_all"
    description                = "Allow HTTP connections from all locations"
    priority                   = 300
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

# ******* STORAGE ACCOUNTS ***********

resource "azurerm_storage_account" "bastion_storage_account" {
  name                = "${var.openshift_cluster_prefix}bsa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type_map["${var.bastion_vm_size}"]}"
}

resource "azurerm_storage_account" "master_storage_account" {
  name                = "${var.openshift_cluster_prefix}msa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type_map["${var.master_vm_size}"]}"
}

resource "azurerm_storage_account" "infra_storage_account" {
  name                = "${var.openshift_cluster_prefix}infrasa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type_map["${var.infra_vm_size}"]}"
}

resource "azurerm_storage_account" "nodeos_storage_account" {
  name                = "${var.openshift_cluster_prefix}nodeossa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type_map["${var.node_vm_size}"]}"
}

resource "azurerm_storage_account" "nodedata_storage_account" {
  name                = "${var.openshift_cluster_prefix}nodedatasa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type_map["${var.node_vm_size}"]}"
}

resource "azurerm_storage_account" "registry_storage_account" {
  name                = "${var.openshift_cluster_prefix}regsa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "Standard_LRS"
}

resource "azurerm_storage_account" "persistent_volume_storage_account" {
  name                = "${var.openshift_cluster_prefix}pvsa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "Standard_LRS"
}

# ******* AVAILABILITY SETS ***********

resource "azurerm_availability_set" "master" {
  name                = "masteravailabilityset"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
}

resource "azurerm_availability_set" "infra" {
  name                = "infraavailabilityset"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
}

resource "azurerm_availability_set" "node" {
  name                = "nodeavailabilityset"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
}

# ******* IP ADDRESSES ***********

resource "azurerm_public_ip" "bastion_pip" {
  name                         = "bastionpip"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  location                     = "${azurerm_resource_group.rg.location}"
  public_ip_address_allocation = "Static"
  domain_name_label            = "${var.openshift_cluster_prefix}-bastion"
}

resource "azurerm_public_ip" "openshift_master_pip" {
  name                         = "masterpip"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  location                     = "${azurerm_resource_group.rg.location}"
  public_ip_address_allocation = "Static"
  domain_name_label            = "${var.openshift_cluster_prefix}"
}

resource "azurerm_public_ip" "infra_lb_pip" {
  name                         = "infraip"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  location                     = "${azurerm_resource_group.rg.location}"
  public_ip_address_allocation = "Static"
  domain_name_label            = "${var.openshift_cluster_prefix}infrapip"
}

# ******* VNETS / SUBNETS ***********

resource "azurerm_virtual_network" "vnet" {
  name                = "openshiftvnet"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  address_space       = ["10.0.0.0/8"]
  depends_on          = ["azurerm_virtual_network.vnet"]
}

resource "azurerm_subnet" "master_subnet" {
  name                 = "mastersubnet"
  virtual_network_name = "${azurerm_virtual_network.vnet.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "10.1.0.0/16"
  depends_on           = ["azurerm_virtual_network.vnet"]
}

resource "azurerm_subnet" "node_subnet" {
  name                 = "nodesubnet"
  virtual_network_name = "${azurerm_virtual_network.vnet.name}"
  resource_group_name  = "${azurerm_resource_group.rg.name}"
  address_prefix       = "10.2.0.0/16"
}

# ******* MASTER LOAD BALANCER ***********

resource "azurerm_lb" "master_lb" {
  name                = "masterloadbalancer"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  depends_on          = ["azurerm_public_ip.openshift_master_pip"]

  frontend_ip_configuration {
    name                 = "LoadBalancerFrontEnd"
    public_ip_address_id = "${azurerm_public_ip.openshift_master_pip.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "master_lb" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  name                = "loadBalancerBackEnd"
  loadbalancer_id     = "${azurerm_lb.master_lb.id}"
  depends_on          = ["azurerm_lb.master_lb"]
}

resource "azurerm_lb_probe" "master_lb" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  loadbalancer_id     = "${azurerm_lb.master_lb.id}"
  name                = "8443Probe"
  port                = 8443
  interval_in_seconds = 5
  number_of_probes    = 2
  protocol            = "Tcp"
  depends_on          = ["azurerm_lb.master_lb"]
}

resource "azurerm_lb_rule" "master_lb" {
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.master_lb.id}"
  name                           = "OpenShiftAdminConsole"
  protocol                       = "Tcp"
  frontend_port                  = 8443
  backend_port                   = 8443
  frontend_ip_configuration_name = "LoadBalancerFrontEnd"
  backend_address_pool_id        = "${azurerm_lb_backend_address_pool.master_lb.id}"
  load_distribution              = "SourceIP"
  idle_timeout_in_minutes        = 30
  probe_id                       = "${azurerm_lb_probe.master_lb.id}"
  enable_floating_ip             = false
  depends_on                     = ["azurerm_lb_probe.master_lb", "azurerm_lb.master_lb", "azurerm_lb_backend_address_pool.master_lb"]
}

resource "azurerm_lb_nat_rule" "master_lb" {
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.master_lb.id}"
  name                           = "${azurerm_lb.master_lb.name}-SSH-${count.index}"
  protocol                       = "Tcp"
  frontend_port                  = "${count.index + 2200}"
  backend_port                   = 22
  frontend_ip_configuration_name = "LoadBalancerFrontEnd"
  count                          = "${var.master_instance_count}"
  depends_on                     = ["azurerm_lb.master_lb"]
}

# ******* INFRA LOAD BALANCER ***********

resource "azurerm_lb" "infra_lb" {
  name                = "infraloadbalancer"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  depends_on          = ["azurerm_public_ip.infra_lb_pip"]

  frontend_ip_configuration {
    name                 = "LoadBalancerFrontEnd"
    public_ip_address_id = "${azurerm_public_ip.infra_lb_pip.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "infra_lb" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  name                = "loadBalancerBackEnd"
  loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
  depends_on          = ["azurerm_lb.infra_lb"]
}

resource "azurerm_lb_probe" "infra_lb_http_probe" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
  name                = "httpProbe"
  port                = 80
  interval_in_seconds = 5
  number_of_probes    = 2
  protocol            = "Tcp"
  depends_on          = ["azurerm_lb.infra_lb"]
}

resource "azurerm_lb_probe" "infra_lb_https_probe" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
  name                = "httpsProbe"
  port                = 443
  interval_in_seconds = 5
  number_of_probes    = 2
  protocol            = "Tcp"
}

resource "azurerm_lb_rule" "infra_lb_http" {
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.infra_lb.id}"
  name                           = "OpenShiftRouterHTTP"
  protocol                       = "Tcp"
  frontend_port                  = 80
  backend_port                   = 80
  frontend_ip_configuration_name = "LoadBalancerFrontEnd"
  backend_address_pool_id        = "${azurerm_lb_backend_address_pool.infra_lb.id}"
  probe_id                       = "${azurerm_lb_probe.infra_lb_http_probe.id}"
  depends_on                     = ["azurerm_lb_probe.infra_lb_http_probe", "azurerm_lb.infra_lb", "azurerm_lb_backend_address_pool.infra_lb"]
}

resource "azurerm_lb_rule" "infra_lb_https" {
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.infra_lb.id}"
  name                           = "OpenShiftRouterHTTPS"
  protocol                       = "Tcp"
  frontend_port                  = 443
  backend_port                   = 443
  frontend_ip_configuration_name = "LoadBalancerFrontEnd"
  backend_address_pool_id        = "${azurerm_lb_backend_address_pool.infra_lb.id}"
  probe_id                       = "${azurerm_lb_probe.infra_lb_https_probe.id}"
  depends_on                     = ["azurerm_lb_probe.infra_lb_https_probe", "azurerm_lb_backend_address_pool.infra_lb"]
}

# ******* NETWORK INTERFACES ***********

resource "azurerm_network_interface" "bastion_nic" {
  name                      = "bastionnic${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.master_nsg.id}"

  ip_configuration {
    name                          = "bastionip${count.index}"
    subnet_id                     = "${azurerm_subnet.master_subnet.id}"
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = "${azurerm_public_ip.bastion_pip.id}"
  }
}

resource "azurerm_network_interface" "master_nic" {
  name                      = "masternic${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.master_nsg.id}"
  count                     = "${var.master_instance_count}"

  ip_configuration {
    name                                    = "masterip${count.index}"
    subnet_id                               = "${azurerm_subnet.master_subnet.id}"
    private_ip_address_allocation           = "Dynamic"
    load_balancer_backend_address_pools_ids = ["${azurerm_lb_backend_address_pool.master_lb.id}"]
    load_balancer_inbound_nat_rules_ids     = ["${element(azurerm_lb_nat_rule.master_lb.*.id, count.index)}"]
  }
}

resource "azurerm_network_interface" "infra_nic" {
  name                      = "infra_nic${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.infra_nsg.id}"
  count                     = "${var.infra_instance_count}"

  ip_configuration {
    name                                    = "infraip${count.index}"
    subnet_id                               = "${azurerm_subnet.master_subnet.id}"
    private_ip_address_allocation           = "Dynamic"
    load_balancer_backend_address_pools_ids = ["${azurerm_lb_backend_address_pool.infra_lb.id}"]
  }
}

resource "azurerm_network_interface" "node_nic" {
  name                      = "node_nic${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.node_nsg.id}"
  count                     = "${var.node_instance_count}"

  ip_configuration {
    name                          = "nodeip${count.index}"
    subnet_id                     = "${azurerm_subnet.node_subnet.id}"
    private_ip_address_allocation = "Dynamic"
  }
}

# ******* Bastion Host *******

resource "azurerm_virtual_machine" "bastion" {
  name                             = "${var.openshift_cluster_prefix}-bastion-1"
  location                         = "${azurerm_resource_group.rg.location}"
  resource_group_name              = "${azurerm_resource_group.rg.name}"
  network_interface_ids            = ["${azurerm_network_interface.bastion_nic.id}"]
  vm_size                          = "${var.bastion_vm_size}"
  delete_os_disk_on_termination    = true
  delete_data_disks_on_termination = true

  tags {
    displayName = "${var.openshift_cluster_prefix}-bastion VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-bastion-${count.index}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = "${var.ssh_public_key}"
    }
  }

  storage_image_reference {
    publisher = "${lookup(var.os_image_map, join("_publisher", list(var.os_image, "")))}"
    offer     = "${lookup(var.os_image_map, join("_offer", list(var.os_image, "")))}"
    sku       = "${lookup(var.os_image_map, join("_sku", list(var.os_image, "")))}"
    version   = "${lookup(var.os_image_map, join("_version", list(var.os_image, "")))}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-master-osdisk${count.index}"
    vhd_uri       = "${azurerm_storage_account.bastion_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-bastion-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
    disk_size_gb  = 60
  }
}

# ******* Master VMs *******

resource "azurerm_virtual_machine" "master" {
  name                             = "${var.openshift_cluster_prefix}-master-${count.index}"
  location                         = "${azurerm_resource_group.rg.location}"
  resource_group_name              = "${azurerm_resource_group.rg.name}"
  availability_set_id              = "${azurerm_availability_set.master.id}"
  network_interface_ids            = ["${element(azurerm_network_interface.master_nic.*.id, count.index)}"]
  vm_size                          = "${var.master_vm_size}"
  delete_os_disk_on_termination    = true
  delete_data_disks_on_termination = true
  count                            = "${var.master_instance_count}"
  depends_on                       = ["azurerm_virtual_machine.infra", "azurerm_virtual_machine.node"]

  tags {
    displayName = "${var.openshift_cluster_prefix}-master VM Creation"
  }

  connection {
    host        = "${azurerm_public_ip.openshift_master_pip.fqdn}"
    user        = "${var.admin_username}"
    port        = 2200
    private_key = "${file(var.connection_private_ssh_key_path)}"
  }

  provisioner "file" {
    source      = "${var.openshift_script_path}/masterPrep.sh"
    destination = "masterPrep.sh"
  }

  provisioner "file" {
    source      = "${var.openshift_script_path}/deployOpenShift.sh"
    destination = "deployOpenShift.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x masterPrep.sh",
      "chmod +x deployOpenShift.sh",
      "sudo bash masterPrep.sh \"${azurerm_storage_account.persistent_volume_storage_account.name}\" \"${var.admin_username}\" && sudo bash deployOpenShift.sh \"${var.admin_username}\" \"${var.openshift_password}\" \"${var.key_vault_secret}\" \"${var.openshift_cluster_prefix}-master\" \"${azurerm_public_ip.openshift_master_pip.fqdn}\" \"${azurerm_public_ip.openshift_master_pip.ip_address}\" \"${var.openshift_cluster_prefix}-infra\" \"${var.openshift_cluster_prefix}-node\" \"${var.node_instance_count}\" \"${var.infra_instance_count}\" \"${var.master_instance_count}\" \"${var.default_sub_domain_type}\" \"${azurerm_storage_account.registry_storage_account.name}\" \"${azurerm_storage_account.registry_storage_account.primary_access_key}\" \"${var.tenant_id}\" \"${var.subscription_id}\" \"${var.aad_client_id}\" \"${var.aad_client_secret}\" \"${azurerm_resource_group.rg.name}\" \"${azurerm_resource_group.rg.location}\" \"${var.key_vault_name}\""
    ]
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-master-${count.index}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = "${var.ssh_public_key}"
    }
  }

  storage_image_reference {
    publisher = "${lookup(var.os_image_map, join("_publisher", list(var.os_image, "")))}"
    offer     = "${lookup(var.os_image_map, join("_offer", list(var.os_image, "")))}"
    sku       = "${lookup(var.os_image_map, join("_sku", list(var.os_image, "")))}"
    version   = "${lookup(var.os_image_map, join("_version", list(var.os_image, "")))}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-master-osdisk${count.index}"
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-master-osdisk${count.index}.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
    disk_size_gb  = 60
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-master-docker-pool${count.index}"
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-master-docker-pool${count.index}.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* Infra VMs *******

resource "azurerm_virtual_machine" "infra" {
  name                             = "${var.openshift_cluster_prefix}-infra-${count.index}"
  location                         = "${azurerm_resource_group.rg.location}"
  resource_group_name              = "${azurerm_resource_group.rg.name}"
  availability_set_id              = "${azurerm_availability_set.infra.id}"
  network_interface_ids            = ["${element(azurerm_network_interface.infra_nic.*.id, count.index)}"]
  vm_size                          = "${var.infra_vm_size}"
  delete_os_disk_on_termination    = true
  delete_data_disks_on_termination = true
  count                            = "${var.infra_instance_count}"

  tags {
    displayName = "${var.openshift_cluster_prefix}-infra VM Creation"
  }

  connection {
    type                = "ssh"
    bastion_host        = "${azurerm_public_ip.bastion_pip.fqdn}"
    bastion_user        = "${var.admin_username}"
    bastion_private_key = "${file(var.connection_private_ssh_key_path)}"
    host                = "${element(azurerm_network_interface.infra_nic.*.private_ip_address, count.index)}"
    user                = "${var.admin_username}"
    private_key         = "${file(var.connection_private_ssh_key_path)}"
  }

  provisioner "file" {
    source      = "${var.openshift_script_path}/nodePrep.sh"
    destination = "nodePrep.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x nodePrep.sh",
      "sudo bash nodePrep.sh",
    ]
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-infra-${count.index}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = "${var.ssh_public_key}"
    }
  }

  storage_image_reference {
    publisher = "${lookup(var.os_image_map, join("_publisher", list(var.os_image, "")))}"
    offer     = "${lookup(var.os_image_map, join("_offer", list(var.os_image, "")))}"
    sku       = "${lookup(var.os_image_map, join("_sku", list(var.os_image, "")))}"
    version   = "${lookup(var.os_image_map, join("_version", list(var.os_image, "")))}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-infra-osdisk${count.index}"
    vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-infra-osdisk${count.index}.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-infra-docker-pool"
    vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-infra-docker-pool${count.index}.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* Node VMs *******

resource "azurerm_virtual_machine" "node" {
  name                             = "${var.openshift_cluster_prefix}-node-${count.index}"
  location                         = "${azurerm_resource_group.rg.location}"
  resource_group_name              = "${azurerm_resource_group.rg.name}"
  availability_set_id              = "${azurerm_availability_set.node.id}"
  network_interface_ids            = ["${element(azurerm_network_interface.node_nic.*.id, count.index)}"]
  vm_size                          = "${var.node_vm_size}"
  delete_os_disk_on_termination    = true
  delete_data_disks_on_termination = true
  count                            = "${var.node_instance_count}"

  tags {
    displayName = "${var.openshift_cluster_prefix}-node VM Creation"
  }

  connection {
    type                = "ssh"
    bastion_host        = "${azurerm_public_ip.bastion_pip.fqdn}"
    bastion_user        = "${var.admin_username}"
    bastion_private_key = "${file(var.connection_private_ssh_key_path)}"
    host                = "${element(azurerm_network_interface.node_nic.*.private_ip_address, count.index)}"
    user                = "${var.admin_username}"
    private_key         = "${file(var.connection_private_ssh_key_path)}"
  }

  provisioner "file" {
    source      = "${var.openshift_script_path}/nodePrep.sh"
    destination = "nodePrep.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x nodePrep.sh",
      "sudo bash nodePrep.sh",
    ]
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-node-${count.index}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = "${var.ssh_public_key}"
    }
  }

  storage_image_reference {
    publisher = "${lookup(var.os_image_map, join("_publisher", list(var.os_image, "")))}"
    offer     = "${lookup(var.os_image_map, join("_offer", list(var.os_image, "")))}"
    sku       = "${lookup(var.os_image_map, join("_sku", list(var.os_image, "")))}"
    version   = "${lookup(var.os_image_map, join("_version", list(var.os_image, "")))}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-node-osdisk"
    vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-node-osdisk${count.index}.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-node-docker-pool${count.index}"
    vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-node-docker-pool${count.index}.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* VM EXTENSIONS *******


# resource "azurerm_virtual_machine_extension" "deploy_open_shift_master" {
#   name                       = "masterOpShExt${count.index}"
#   location                   = "${azurerm_resource_group.rg.location}"
#   resource_group_name        = "${azurerm_resource_group.rg.name}"
#   virtual_machine_name       = "${element(azurerm_virtual_machine.master.*.name, count.index)}"
#   publisher                  = "Microsoft.Azure.Extensions"
#   type                       = "CustomScript"
#   type_handler_version       = "2.0"
#   auto_upgrade_minor_version = true
#   depends_on                 = ["azurerm_virtual_machine.master", "azurerm_virtual_machine_extension.node_prep", "azurerm_storage_container.vhds", "azurerm_virtual_machine_extension.deploy_infra"]
#
#   settings = <<SETTINGS
# {
#   "fileUris": [
# 		"${var.artifacts_location}scripts/masterPrep.sh",
#     "${var.artifacts_location}scripts/deployOpenShift.sh"
# 	]
# }
# SETTINGS
#
#   protected_settings = <<SETTINGS
#  {
#    "commandToExecute": "bash masterPrep.sh ${azurerm_storage_account.persistent_volume_storage_account.name} ${var.admin_username} && bash deployOpenShift.sh \"${var.admin_username}\" '${var.openshift_password}' \"${var.key_vault_secret}\" \"${var.openshift_cluster_prefix}-master\" \"${azurerm_public_ip.openshift_master_pip.fqdn}\" \"${azurerm_public_ip.openshift_master_pip.ip_address}\" \"${var.openshift_cluster_prefix}-infra\" \"${var.openshift_cluster_prefix}-node\" \"${var.node_instance_count}\" \"${var.infra_instance_count}\" \"${var.master_instance_count}\" \"${var.default_sub_domain_type}\" \"${azurerm_storage_account.registry_storage_account.name}\" \"${azurerm_storage_account.registry_storage_account.primary_access_key}\" \"${var.tenant_id}\" \"${var.subscription_id}\" \"${var.aad_client_id}\" \"${var.aad_client_secret}\" \"${azurerm_resource_group.rg.name}\" \"${azurerm_resource_group.rg.location}\" \"${var.key_vault_name}\""
#  }
# SETTINGS
# }


# resource "azurerm_virtual_machine_extension" "deploy_infra" {
#   name                       = "infraOpShExt${count.index}"
#   location                   = "${azurerm_resource_group.rg.location}"
#   resource_group_name        = "${azurerm_resource_group.rg.name}"
#   virtual_machine_name       = "${element(azurerm_virtual_machine.infra.*.name, count.index)}"
#   publisher                  = "Microsoft.Azure.Extensions"
#   type                       = "CustomScript"
#   type_handler_version       = "2.0"
#   auto_upgrade_minor_version = true
#   depends_on                 = ["azurerm_virtual_machine.infra"]
#
#   settings = <<SETTINGS
# {
#   "fileUris": [
# 		"${var.artifacts_location}scripts/nodePrep.sh"
# 	]
# }
# SETTINGS
#
#   protected_settings = <<SETTINGS
# {
# 	"commandToExecute": "bash nodePrep.sh"
# }
# SETTINGS
# }


# resource "azurerm_virtual_machine_extension" "node_prep" {
#   name                       = "nodePrepExt${count.index}"
#   location                   = "${azurerm_resource_group.rg.location}"
#   resource_group_name        = "${azurerm_resource_group.rg.name}"
#   virtual_machine_name       = "${element(azurerm_virtual_machine.node.*.name, count.index)}"
#   publisher                  = "Microsoft.Azure.Extensions"
#   type                       = "CustomScript"
#   type_handler_version       = "2.0"
#   auto_upgrade_minor_version = true
#   depends_on                 = ["azurerm_virtual_machine.node", "azurerm_storage_account.nodeos_storage_account"]
#
#   settings = <<SETTINGS
# {
#   "fileUris": [
# 		"${var.artifacts_location}scripts/nodePrep.sh"
# 	]
# }
# SETTINGS
#
#   protected_settings = <<SETTINGS
# {
# 	"commandToExecute": "bash nodePrep.sh"
# }
# SETTINGS
# }

