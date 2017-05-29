# provider "azurerm" {
#   subscription_id = "REPLACE-WITH-YOUR-SUBSCRIPTION-ID"
#   client_id       = "REPLACE-WITH-YOUR-CLIENT-ID"
#   client_secret   = "REPLACE-WITH-YOUR-CLIENT-SECRET"
#   tenant_id       = "REPLACE-WITH-YOUR-TENANT-ID"
# }

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

resource "azurerm_network_interface" "master_nic" {
  name                      = "masternic${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.master_nsg.id}"
  count                     = "${var.master_instance_count}"
  depends_on                = ["azurerm_subnet.master_subnet", "azurerm_lb.master_lb", "azurerm_lb_backend_address_pool.master_lb"]

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
  depends_on                = ["azurerm_subnet.master_subnet", "azurerm_lb.infra_lb", "azurerm_lb_backend_address_pool.infra_lb", "azurerm_network_security_group.infra_nsg"]

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
  depends_on                = ["azurerm_subnet.node_subnet", "azurerm_network_security_group.node_nsg"]

  ip_configuration {
    name                          = "nodeip${count.index}"
    subnet_id                     = "${azurerm_subnet.node_subnet.id}"
    private_ip_address_allocation = "Dynamic"
  }
}

# ******* Master VMs *******

resource "azurerm_virtual_machine" "master" {
  name                  = "masterVm${count.index}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.master.id}"
  network_interface_ids = ["${element(azurerm_network_interface.master_nic.*.id, count.index)}"]
  vm_size               = "${var.master_vm_size}"
  count                 = "${var.master_instance_count}"
  depends_on            = ["azurerm_network_interface.master_nic", "azurerm_availability_set.master"]

  tags {
    displayName = "${var.openshift_cluster_prefix}-master VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-master${count.index}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = "${file(var.ssh_public_key_path)}"
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
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-master-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-master-docker-pool${count.index}"
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-master-docker-pool.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* Infra VMs *******

resource "azurerm_virtual_machine" "infra" {
  name                  = "infraVm${count.index}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.infra.id}"
  network_interface_ids = ["${element(azurerm_network_interface.infra_nic.*.id, count.index)}"]
  vm_size               = "${var.infra_vm_size}"
  count                 = "${var.infra_instance_count}"
  depends_on            = ["azurerm_network_interface.infra_nic"]
  depends_on            = ["azurerm_availability_set.infra"]

  tags {
    displayName = "${var.openshift_cluster_prefix}-infra VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-infra${count.index}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = "${file(var.ssh_public_key_path)}"
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
    vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-infra-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-infra-docker-pool"
    vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-infra-docker-pool.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# # ******* Node VMs *******

resource "azurerm_virtual_machine" "node" {
  name                  = "nodeVm${count.index}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.node.id}"
  network_interface_ids = ["${element(azurerm_network_interface.node_nic.*.id, count.index)}"]
  vm_size               = "${var.node_vm_size}"
  count                 = "${var.node_instance_count}"
  depends_on            = ["azurerm_network_interface.node_nic"]
  depends_on            = ["azurerm_availability_set.node"]

  tags {
    displayName = "${var.openshift_cluster_prefix}-node VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-node${count.index}"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = true

    ssh_keys {
      path     = "/home/${var.admin_username}/.ssh/authorized_keys"
      key_data = "${file(var.ssh_public_key_path)}"
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
    vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-node-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-node-docker-pool${count.index}"
    vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-node-docker-pool.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* VM EXTENSIONS *******

resource "azurerm_virtual_machine_extension" "deploy_open_shift_master" {
  name                       = "masterOpShExt${count.index}"
  location                   = "${azurerm_resource_group.rg.location}"
  resource_group_name        = "${azurerm_resource_group.rg.name}"
  virtual_machine_name       = "${element(azurerm_virtual_machine.master.*.name, count.index)}"
  publisher                  = "Microsoft.Azure.Extensions"
  type                       = "CustomScript"
  type_handler_version       = "2.0"
  auto_upgrade_minor_version = true
  depends_on                 = ["azurerm_virtual_machine.master"]

  settings = <<SETTINGS
{
  "fileUris": [
		"${var.artifacts_location}scripts/masterPrep.sh"
	]
}
SETTINGS

  protected_settings = <<SETTINGS
{
	"commandToExecute": "bash masterPrep.sh ${azurerm_storage_account.persistent_volume_storage_account.name} ${var.admin_username}"
}
SETTINGS
}

resource "azurerm_virtual_machine_extension" "deploy_infra" {
  name                       = "infraOpShExt${count.index}"
  location                   = "${azurerm_resource_group.rg.location}"
  resource_group_name        = "${azurerm_resource_group.rg.name}"
  virtual_machine_name       = "${element(azurerm_virtual_machine.infra.*.name, count.index)}"
  publisher                  = "Microsoft.Azure.Extensions"
  type                       = "CustomScript"
  type_handler_version       = "2.0"
  auto_upgrade_minor_version = true
  depends_on                 = ["azurerm_virtual_machine.infra"]

  settings = <<SETTINGS
{
  "fileUris": [
		"${var.artifacts_location}scripts/nodePrep.sh"
	]
}
SETTINGS

  protected_settings = <<SETTINGS
{
	"commandToExecute": "bash nodePrep.sh"
}
SETTINGS
}

resource "azurerm_virtual_machine_extension" "deploy_nodes" {
  name                       = "nodePrepExt${count.index}"
  location                   = "${azurerm_resource_group.rg.location}"
  resource_group_name        = "${azurerm_resource_group.rg.name}"
  virtual_machine_name       = "${element(azurerm_virtual_machine.node.*.name, count.index)}"
  publisher                  = "Microsoft.Azure.Extensions"
  type                       = "CustomScript"
  type_handler_version       = "2.0"
  auto_upgrade_minor_version = true
  depends_on                 = ["azurerm_virtual_machine.node", "azurerm_storage_accounts.node"]

  settings = <<SETTINGS
{
  "fileUris": [
		"${var.artifacts_location}scripts/nodePrep.sh"
	]
}
SETTINGS

  protected_settings = <<SETTINGS
{
	"commandToExecute": "bash nodePrep.sh"
}
SETTINGS
}

# resource "azurerm_template_deployment" "test" {
#   name                = "OpenShiftDeployment"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   depends_on                 = ["azurerm_virtual_machine.master", "azurerm_virtual_machine.infra", "azurerm_virtual_machine.node", "azurerm_storage_account.persistent_volume_storage_account", "azurerm_storage_account.registry_storage_account"]


#   template_body = <<DEPLOY
# 	  "properties": {
# 			"mode": "Incremental",
# 			"templateLink": {
# 				"uri": "${var.artifacts_location}scripts/deployOpenShift.sh",
# 				"contentVersion": "1.0.0.0"
# 			},
# 			"parameters": {
# 				"_artifactsLocation": {
# 					"value": "${var.artifacts_location}"
# 				},
# 				"apiVersionCompute": {
# 					"value": "${var.api_version_compute}"
# 				},
# 				"newStorageAccountRegistry": {
# 					"value": "${azurerm_storage_account.registry_storage_account.name}"
# 				},
# 				"newStorageAccountKey": {
# 					"value": "${azurerm_storage_account.registry_storage_account.primary_access_key}"
# 				},
# 				"newStorageAccountPersistentVolume1": {
# 					"value": "${azurerm_storage_account.persistent_volume_storage_account.name}"
# 				},
# 				"newStorageAccountPV1Key": {
# 					"value": "${azurerm_storage_account.persistent_volume_storage_account.primary_access_key}"
# 				},
# 				"openshiftMasterHostname": {
# 					"value": "[variables('openshiftMasterHostname')]"
# 				},
# 				"openshiftMasterPublicIpFqdn": {
# 					"value": "${var.azurerm_public_ip.openshift_master_pip.fqdn}"
# 				},
# 				"openshiftMasterPublicIpAddress": {
# 					"value": "${var.azurerm_public_ip.openshift_master_pip.ip_address}"
# 				},
# 				"openshiftInfraHostname": {
# 					"value": "[variables('openshiftInfraHostname')]"
# 				},
# 				"openshiftNodeHostname": {
# 					"value": "[variables('openshiftNodeHostname')]"
# 				},
# 				"masterInstanceCount": {
# 					"value": "${var.master_instance_count}"
# 				},
# 				"infraInstanceCount": {
# 					"value": "${var.infra_instance_count}"
# 				},
# 				"nodeInstanceCount": {
# 					"value": "${var.node_instance_count}"
# 				},
# 				"adminUsername": {
# 					"value": "${var.admin_username}"
# 				},
# 				"openshiftPassword": {
# 					"value": "${var.openshift_password}"
# 				},
# 				"aadClientId": {
# 					"value": "${var.aad_client_id}"
# 				},
# 				"aadClientSecret": {
# 					"value": "${var.aad_client_secret}"
# 				},
# 				"xipioDomain": {
# 					"value": "${azurerm_public_ip.infra_lb_pip.ip_address}.xip.io"
# 				},
# 				"customDomain": {
# 					"value": "${var.default_sub_domain}"
# 				},
# 				"subDomainChosen": {
# 					"value": "${var.default_sub_domain_type} Domain"
# 				},
# 				"sshPrivateKey": {
# 					"reference": {
# 						"keyvault": {
# 							"id": "/subscriptions/${var.subscription_id}/resourceGroups/${var.key_vault_resource_group}/providers/Microsoft.KeyVault/vaults/${var.key_vault_name}"
# 						},
# 						"secretName": "${var.key_vault_secret}"
# 					}
# 				}
# 			}
# 		}
# 	}
# DEPLOY
# "outputs": {
# 	"Openshift Console Url": {
# 		"type": "string",
# 		"value": "https://${azurerm_public_ip.openshift_master_pip.fqdn}:8443/console"
# 	},
# 	"Openshift Master SSH": {
# 		"type": "string",
# 		"value": "ssh ${var.admin_username}@${azurerm_public_ip.openshift_master_pip.fqdn} -p 2200"
# 	},
# 	"Openshift Infra Load Balancer FQDN": {
# 		"type": "string",
# 		"value": "${azurerm_public_ip.infra_lb_pip.fqdn}"
# 	},
# 	"Node OS Storage Account Name": {
# 		"type": "string",
# 		"value": "${azurerm_storage_account.nodeos_storage_account.name}"
# 	},
# 	"Node Data Storage Account Name": {
# 		"type": "string",
# 		"value": "${azurerm_storage_account.nodedata_storage_account.name}"
# 	},
# 	"Infra Storage Account Name": {
# 		"type": "string",
# 		"value": "${azurerm_storage_account.infra_storage_account.name}"
# 	}
# }

