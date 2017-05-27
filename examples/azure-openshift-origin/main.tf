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

# resource "azurerm_network_security_group" "infra_nsg" {
#   name                = "${var.openshift_cluster_prefix}-infra-nsg"
#   location            = "${azurerm_resource_group.rg.location}"
#   resource_group_name = "${azurerm_resource_group.rg.name}"

#   security_rule {
#     name                       = "allow_SSH_in_all"
#     description                = "Allow SSH in from all locations"
#     priority                   = 100
#     direction                  = "Inbound"
#     access                     = "Allow"
#     protocol                   = "Tcp"
#     source_port_range          = "*"
#     destination_port_range     = "22"
#     source_address_prefix      = "*"
#     destination_address_prefix = "*"
#   }

#   security_rule {
#     name                       = "allow_HTTPS_all"
#     description                = "Allow HTTPS connections from all locations"
#     priority                   = 200
#     direction                  = "Inbound"
#     access                     = "Allow"
#     protocol                   = "Tcp"
#     source_port_range          = "*"
#     destination_port_range     = "443"
#     source_address_prefix      = "*"
#     destination_address_prefix = "*"
#   }

#   security_rule {
#     name                       = "allow_HTTP_in_all"
#     description                = "Allow HTTP connections from all locations"
#     priority                   = 300
#     direction                  = "Inbound"
#     access                     = "Allow"
#     protocol                   = "Tcp"
#     source_port_range          = "*"
#     destination_port_range     = "80"
#     source_address_prefix      = "*"
#     destination_address_prefix = "*"
#   }
# }

# resource "azurerm_network_security_group" "node_nsg" {
#   name                = "${var.openshift_cluster_prefix}-node-nsg"
#   location            = "${azurerm_resource_group.rg.location}"
#   resource_group_name = "${azurerm_resource_group.rg.name}"

#   security_rule {
#     name                       = "allow_SSH_in_all"
#     description                = "Allow SSH in from all locations"
#     priority                   = 100
#     direction                  = "Inbound"
#     access                     = "Allow"
#     protocol                   = "Tcp"
#     source_port_range          = "*"
#     destination_port_range     = "22"
#     source_address_prefix      = "*"
#     destination_address_prefix = "*"
#   }

#   security_rule {
#     name                       = "allow_HTTPS_all"
#     description                = "Allow HTTPS connections from all locations"
#     priority                   = 200
#     direction                  = "Inbound"
#     access                     = "Allow"
#     protocol                   = "Tcp"
#     source_port_range          = "*"
#     destination_port_range     = "443"
#     source_address_prefix      = "*"
#     destination_address_prefix = "*"
#   }

#   security_rule {
#     name                       = "allow_HTTP_in_all"
#     description                = "Allow HTTP connections from all locations"
#     priority                   = 300
#     direction                  = "Inbound"
#     access                     = "Allow"
#     protocol                   = "Tcp"
#     source_port_range          = "*"
#     destination_port_range     = "80"
#     source_address_prefix      = "*"
#     destination_address_prefix = "*"
#   }
# }

# ******* STORAGE ACCOUNTS ***********

resource "azurerm_storage_account" "master_storage_account" {
  name                = "${var.openshift_cluster_prefix}msa"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"
  account_type        = "${var.storage_account_type_map["${var.master_vm_size}"]}"
}

# resource "azurerm_storage_account" "infra_storage_account" {
#   name                = "${var.openshift_cluster_prefix}infrasa"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   location            = "${azurerm_resource_group.rg.location}"
#   account_type        = "${var.storage_account_type_map["${var.infra_vm_size}"]}"
# }

# resource "azurerm_storage_account" "nodeos_storage_account" {
#   name                = "${var.openshift_cluster_prefix}nodeossa"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   location            = "${azurerm_resource_group.rg.location}"
#   account_type        = "${var.storage_account_type_map["${var.node_vm_size}"]}"
# }

# resource "azurerm_storage_account" "nodedata_storage_account" {
#   name                = "${var.openshift_cluster_prefix}nodedatasa"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   location            = "${azurerm_resource_group.rg.location}"
#   account_type        = "${var.storage_account_type_map["${var.node_vm_size}"]}"
# }

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

# resource "azurerm_availability_set" "infra" {
#   name                = "infraavailabilityset"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   location            = "${azurerm_resource_group.rg.location}"
# }

# resource "azurerm_availability_set" "node" {
#   name                = "nodeavailabilityset"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   location            = "${azurerm_resource_group.rg.location}"
# }

# ******* IP ADDRESSES ***********

resource "azurerm_public_ip" "openshift_master_pip" {
  name                         = "masterpip${count.index}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  location                     = "${azurerm_resource_group.rg.location}"
  public_ip_address_allocation = "Static"
  domain_name_label            = "${var.openshift_cluster_prefix}masterpip${count.index}"
}

# resource "azurerm_public_ip" "infra_lb_pip" {
#   name                         = "${var.infra_lb_publicip_dns_label}"
#   resource_group_name          = "${azurerm_resource_group.rg.name}"
#   location                     = "${azurerm_resource_group.rg.location}"
#   public_ip_address_allocation = "Static"
#   domain_name_label            = "${var.infra_lb_publicip_dns_label}infrapip"
# }

# ******* VNETS / SUBNETS ***********

resource "azurerm_virtual_network" "vnet" {
  name                = "openshiftvnet"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  address_space       = ["10.0.0.0/8"]
}

resource "azurerm_subnet" "master_subnet" {
  name                      = "mastersubnet"
  virtual_network_name      = "${azurerm_virtual_network.vnet.name}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  address_prefix            = "10.1.0.0/16"
}

# resource "azurerm_subnet" "node_subnet" {
#   name                      = "nodesubnet"
#   virtual_network_name      = "${azurerm_virtual_network.vnet.name}"
#   resource_group_name       = "${azurerm_resource_group.rg.name}"
#   address_prefix            = "10.2.0.0/16"
#   # network_security_group_id = "${azurerm_network_security_group.node_nsg.id}"
# }

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
  depends_on                     = ["azurerm_lb_probe.master_lb"]
  depends_on                     = ["azurerm_lb.master_lb"]
  depends_on                     = ["azurerm_lb_backend_address_pool.master_lb"]
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

# resource "azurerm_lb_nat_rule" "master_lb_443" {
#   resource_group_name            = "${azurerm_resource_group.rg.name}"
#   loadbalancer_id                = "${azurerm_lb.master_lb.id}"
#   name                           = "${azurerm_lb.master_lb.name}-443-${count.index}"
#   protocol                       = "Tcp"
#   frontend_port                  = 443
#   backend_port                   = 443
#   frontend_ip_configuration_name = "LoadBalancerFrontEnd"
#   count                          = "${var.master_instance_count}"
#   depends_on                     = ["azurerm_lb.master_lb"]
# }

# resource "azurerm_lb_nat_rule" "master_lb_80" {
#   resource_group_name            = "${azurerm_resource_group.rg.name}"
#   loadbalancer_id                = "${azurerm_lb.master_lb.id}"
#   name                           = "${azurerm_lb.master_lb.name}-80-${count.index}"
#   protocol                       = "Tcp"
#   frontend_port                  = 80
#   backend_port                   = 80
#   frontend_ip_configuration_name = "LoadBalancerFrontEnd"
#   count                          = "${var.master_instance_count}"
#   depends_on                     = ["azurerm_lb.master_lb"]
# }

# # ******* INFRA LOAD BALANCER ***********

# resource "azurerm_lb" "infra_lb" {
#   name                = "infraloadbalancer"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   location            = "${azurerm_resource_group.rg.location}"

#   frontend_ip_configuration {
#     name                 = "LoadBalancerFrontEnd"
#     public_ip_address_id = "${azurerm_public_ip.infra_lb_pip.id}"
#   }
# }

# resource "azurerm_lb_backend_address_pool" "infra_lb" {
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   name                = "loadBalancerBackEnd"
#   loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
# }

# resource "azurerm_lb_probe" "infra_lb_http_probe" {
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
#   name                = "httpProbe"
#   port                = 80
#   interval_in_seconds = 5
#   number_of_probes    = 2
#   protocol            = "Tcp"
# }

# resource "azurerm_lb_probe" "infra_lb_https_probe" {
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
#   name                = "httpsProbe"
#   port                = 443
#   interval_in_seconds = 5
#   number_of_probes    = 2
#   protocol            = "Tcp"
# }

# resource "azurerm_lb_rule" "infra_lb_http" {
#   resource_group_name            = "${azurerm_resource_group.rg.name}"
#   loadbalancer_id                = "${azurerm_lb.infra_lb.id}"
#   name                           = "OpenShiftRouterHTTP"
#   protocol                       = "Tcp"
#   frontend_port                  = 80
#   backend_port                   = 80
#   frontend_ip_configuration_name = "LoadBalancerFrontEnd"
#   backend_address_pool_id        = "${azurerm_lb_backend_address_pool.infra_lb.id}"
#   idle_timeout_in_minutes        = 30
#   probe_id                       = "${azurerm_lb_probe.infra_lb_http_probe.id}"
# }

# resource "azurerm_lb_rule" "infra_lb_https" {
#   resource_group_name            = "${azurerm_resource_group.rg.name}"
#   loadbalancer_id                = "${azurerm_lb.infra_lb.id}"
#   name                           = "OpenShiftRouterHTTPS"
#   protocol                       = "Tcp"
#   frontend_port                  = 443
#   backend_port                   = 443
#   frontend_ip_configuration_name = "LoadBalancerFrontEnd"
#   backend_address_pool_id        = "${azurerm_lb_backend_address_pool.infra_lb.id}"
#   idle_timeout_in_minutes        = 30
#   probe_id                       = "${azurerm_lb_probe.infra_lb_https_probe.id}"
# }

# ******* NETWORK INTERFACES ***********

resource "azurerm_network_interface" "master_nic" {
  name                      = "masternic${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.master_nsg.id}"
  count                     = "${var.master_instance_count}"
  depends_on                = ["azurerm_subnet.master_subnet"]
  depends_on                = ["azurerm_lb_nat_rule.master_lb"]
  depends_on                = ["azurerm_lb_backend_address_pool.master_lb"]

  ip_configuration {
    name                                    = "masteripconfig${count.index}"
    subnet_id                               = "${azurerm_subnet.master_subnet.id}"
    private_ip_address_allocation           = "Dynamic"
    load_balancer_backend_address_pools_ids = ["${azurerm_lb_backend_address_pool.master_lb.id}"]
    load_balancer_inbound_nat_rules_ids     = ["${element(azurerm_lb_nat_rule.master_lb.*.id, count.index)}"]
  }
}

# resource "azurerm_network_interface" "infra_nic" {
#   name                      = "infra_nic${count.index}"
#   location                  = "${azurerm_resource_group.rg.location}"
#   resource_group_name       = "${azurerm_resource_group.rg.name}"
#   network_security_group_id = "${azurerm_network_security_group.infra_nsg.id}"
#   count                     = "${var.infra_instance_count}"

#   ip_configuration {
#     name                                    = "infraipconfig${count.index}"
#     subnet_id                               = "${azurerm_subnet.master_subnet.id}"
#     private_ip_address_allocation           = "Dynamic"
#     load_balancer_backend_address_pools_ids = ["${azurerm_lb_backend_address_pool.infra_lb.id}"]
#   }
# }

# resource "azurerm_network_interface" "node_nic" {
#   name                      = "node_nic${count.index}"
#   location                  = "${azurerm_resource_group.rg.location}"
#   resource_group_name       = "${azurerm_resource_group.rg.name}"
#   network_security_group_id = "${azurerm_network_security_group.node_nsg.id}"
#   count                     = "${var.node_instance_count}"

#   ip_configuration {
#     name                          = "nodeipconfig${count.index}"
#     subnet_id                     = "${azurerm_subnet.node_subnet.id}"
#     private_ip_address_allocation = "Dynamic"
#   }
# }

# ******* Master VMs *******

resource "azurerm_virtual_machine" "master" {
  name                  = "masterVm${count.index}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.master.id}"
  network_interface_ids = ["${element(azurerm_network_interface.master_nic.*.id, count.index)}"]
  vm_size               = "${var.master_vm_size}"
  count                 = "${var.master_instance_count}"
  depends_on            = ["azurerm_network_interface.master_nic"]
  depends_on            = ["azurerm_availability_set.master"]

  tags {
    displayName = "${var.openshift_cluster_prefix}-master VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-master"
    admin_username = "${var.admin_username}"
    admin_password = "${var.openshift_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false

    # ssh_keys {
    #   path     = "/home/${var.admin_username}/.ssh/authorized_keys"
    #   key_data = "${var.ssh_public_key}"
    # }
  }

  storage_image_reference {
    publisher = "${lookup(var.os_image_map, join("_publisher", list(var.os_image, "")))}"
    offer     = "${lookup(var.os_image_map, join("_offer", list(var.os_image, "")))}"
    sku       = "${lookup(var.os_image_map, join("_sku", list(var.os_image, "")))}"
    version   = "${lookup(var.os_image_map, join("_version", list(var.os_image, "")))}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-master-osdisk"
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-master-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-master-docker-pool"
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-master-docker-pool.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* Infra VMs *******

# resource "azurerm_virtual_machine" "infra" {
#   name                  = "infraVm${count.index}"
#   location              = "${azurerm_resource_group.rg.location}"
#   resource_group_name   = "${azurerm_resource_group.rg.name}"
#   availability_set_id   = "${azurerm_availability_set.infra.id}"
#   network_interface_ids = ["${element(azurerm_network_interface.infra_nic.*.id, count.index)}"]
#   vm_size               = "${var.infra_vm_size}"
#   count                 = "${var.infra_instance_count}"

#   tags {
#     displayName = "${var.openshift_cluster_prefix}-infra VM Creation"
#   }

#   os_profile {
#     computer_name  = "${var.openshift_cluster_prefix}-infra"
#     admin_username = "${var.admin_username}"
#     admin_password = "${var.openshift_password}"
#   }

#   os_profile_linux_config {
#     disable_password_authentication = false

#     # ssh_keys {
#     #   path     = "/home/annie/.ssh/authorized_keys"
#     #   key_data = "${var.ssh_public_key}"
#     # }
#   }

#   storage_image_reference {
#     publisher = "${lookup(var.os_image_map, join("_publisher", list(var.os_image, "")))}"
#     offer     = "${lookup(var.os_image_map, join("_offer", list(var.os_image, "")))}"
#     sku       = "${lookup(var.os_image_map, join("_sku", list(var.os_image, "")))}"
#     version   = "${lookup(var.os_image_map, join("_version", list(var.os_image, "")))}"
#   }

#   storage_os_disk {
#     name          = "${var.openshift_cluster_prefix}-infra-osdisk"
#     vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-infra-osdisk.vhd"
#     caching       = "ReadWrite"
#     create_option = "FromImage"
#   }

#   storage_data_disk {
#     name          = "${var.openshift_cluster_prefix}-infra-docker-pool"
#     vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-infra-docker-pool.vhd"
#     disk_size_gb  = "${var.data_disk_size}"
#     create_option = "Empty"
#     lun           = 0
#   }
# }

# # ******* Node VMs *******

# resource "azurerm_virtual_machine" "node" {
#   name                  = "nodeVm${count.index}"
#   location              = "${azurerm_resource_group.rg.location}"
#   resource_group_name   = "${azurerm_resource_group.rg.name}"
#   availability_set_id   = "${azurerm_availability_set.node.id}"
#   network_interface_ids = ["${element(azurerm_network_interface.node_nic.*.id, count.index)}"]
#   vm_size               = "${var.node_vm_size}"
#   count                 = "${var.node_instance_count}"

#   tags {
#     displayName = "${var.openshift_cluster_prefix}-node VM Creation"
#   }

#   os_profile {
#     computer_name  = "${var.openshift_cluster_prefix}-node"
#     admin_username = "${var.admin_username}"
#     admin_password = "${var.openshift_password}"
#   }

#   os_profile_linux_config {
#     disable_password_authentication = false

#     # ssh_keys {
#     #   path     = "/home/${var.admin_username}/.ssh/authorized_keys"
#     #   key_data = "${var.ssh_public_key}"
#     # }
#   }

#   storage_image_reference {
#     publisher = "${lookup(var.os_image_map, join("_publisher", list(var.os_image, "")))}"
#     offer     = "${lookup(var.os_image_map, join("_offer", list(var.os_image, "")))}"
#     sku       = "${lookup(var.os_image_map, join("_sku", list(var.os_image, "")))}"
#     version   = "${lookup(var.os_image_map, join("_version", list(var.os_image, "")))}"
#   }

#   storage_os_disk {
#     name          = "${var.openshift_cluster_prefix}-node-osdisk"
#     vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-node-osdisk.vhd"
#     caching       = "ReadWrite"
#     create_option = "FromImage"
#   }

#   storage_data_disk {
#     name          = "${var.openshift_cluster_prefix}-node-docker-pool"
#     vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}vhds/${var.openshift_cluster_prefix}-node-docker-pool.vhd"
#     disk_size_gb  = "${var.data_disk_size}"
#     create_option = "Empty"
#     lun           = 0
#   }
# }

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
		"${var.master_artifacts_location}scripts/masterPrep.sh"
	]
}
SETTINGS

  protected_settings = <<SETTINGS
{
	"commandToExecute": "bash masterPrep.sh ${azurerm_storage_account.persistent_volume_storage_account.name} ${var.admin_username}"
}
SETTINGS

}

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

#   settings = <<SETTINGS
# 			"fileUris": [
# 						"${var.artifacts_location}scripts/nodePrep.sh"
# 					]
# 				}
# SETTINGS

#   settings = <<SETTINGS
#     {
# 			"commandToExecute": "bash nodePrep.sh"
# 		}
# SETTINGS
# }

# resource "azurerm_virtual_machine_extension" "deploy_nodes" {
#   name                       = "nodeVmDeployment${count.index}"
#   location                   = "${azurerm_resource_group.rg.location}"
#   resource_group_name        = "${azurerm_resource_group.rg.name}"
#   virtual_machine_name       = "${element(azurerm_virtual_machine.node.*.name, count.index)}"
#   publisher                  = "Microsoft.Azure.Extensions"
#   type                       = "CustomScript"
#   type_handler_version       = "2.0"
#   auto_upgrade_minor_version = true
#   depends_on                 = ["azurerm_virtual_machine.node"]

#   settings = <<SETTINGS
# 			"fileUris": [
# 						"${var.artifacts_location}scripts/nodePrep.sh"
# 					]
# 				}
# SETTINGS

#   settings = <<SETTINGS
#     {
# 			"commandToExecute": "bash nodePrep.sh"
# 		}
# SETTINGS
# }

# resource "azurerm_template_deployment" "test" {
#   name                = "OpenShiftDeployment"
#   resource_group_name = "${azurerm_resource_group.rg.name}"
#   depends_on                 = ["azurerm_virtual_machine.master", "azurerm_virtual_machine.infra", "azurerm_virtual_machine.node"]


#   template_body = <<DEPLOY
# 	  "properties": {
# 			"mode": "Incremental",
# 			"templateLink": {
# 				"uri": "[variables('openshiftDeploymentTemplateUrl')]",
# 				"contentVersion": "1.0.0.0"
# 			},
# 			"parameters": {
# 				"_artifactsLocation": {
# 					"value": "[parameters('_artifactsLocation')]"
# 				},
# 				"apiVersionCompute": {
# 					"value": "[variables('apiVersionCompute')]"
# 				},
# 				"newStorageAccountRegistry": {
# 					"value": "[variables('newStorageAccountRegistry')]"
# 				},
# 				"newStorageAccountKey": {
# 					"value": "[listKeys(variables('newStorageAccountRegistry'),'2015-06-15').key1]"
# 				},
# 				"newStorageAccountPersistentVolume1": {
# 					"value": "[variables('newStorageAccountPersistentVolume1')]"
# 				},
# 				"newStorageAccountPV1Key": {
# 					"value": "[listKeys(variables('newStorageAccountPersistentVolume1'),'2015-06-15').key1]"
# 				},
# 				"openshiftMasterHostname": {
# 					"value": "[variables('openshiftMasterHostname')]"
# 				},
# 				"openshiftMasterPublicIpFqdn": {
# 					"value": "[reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn]"
# 				},
# 				"openshiftMasterPublicIpAddress": {
# 					"value": "[reference(parameters('openshiftMasterPublicIpDnsLabel')).ipAddress]"
# 				},
# 				"openshiftInfraHostname": {
# 					"value": "[variables('openshiftInfraHostname')]"
# 				},
# 				"openshiftNodeHostname": {
# 					"value": "[variables('openshiftNodeHostname')]"
# 				},
# 				"masterInstanceCount": {
# 					"value": "[parameters('masterInstanceCount')]"
# 				},
# 				"infraInstanceCount": {
# 					"value": "[parameters('infraInstanceCount')]"
# 				},
# 				"nodeInstanceCount": {
# 					"value": "[parameters('nodeInstanceCount')]"
# 				},
# 				"adminUsername": {
# 					"value": "[parameters('adminUsername')]"
# 				},
# 				"openshiftPassword": {
# 					"value": "[parameters('openshiftPassword')]"
# 				},
# 				"aadClientId": {
# 					"value": "[parameters('aadClientId')]"
# 				},
# 				"aadClientSecret": {
# 					"value": "[parameters('aadClientSecret')]"
# 				},
# 				"xipioDomain": {
# 					"value": "[concat(reference(parameters('infraLbPublicIpDnsLabel')).ipAddress, '.xip.io')]"
# 				},
# 				"customDomain": {
# 					"value": "[parameters('defaultSubDomain')]"
# 				},
# 				"subDomainChosen": {
# 					"value": "[concat(parameters('defaultSubDomainType'), 'Domain')]"
# 				},
# 				"sshPrivateKey": {
# 					"reference": {
# 						"keyvault": {
# 							"id": "[concat('/subscriptions/', subscription().subscriptionId, '/resourceGroups/', parameters('keyVaultResourceGroup'), '/providers/Microsoft.KeyVault/vaults/', parameters('keyVaultName'))]"
# 						},
# 						"secretName": "[parameters('keyVaultSecret')]"
# 					}
# 				}
# 			}
# 		}
# 	}
# DEPLOY
	# "outputs": {
	# 	"Openshift Console Url": {
	# 		"type": "string",
	# 		"value": "[concat('https://', reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn, ':8443/console')]"
	# 	},
	# 	"Openshift Master SSH": {
	# 		"type": "string",
	# 		"value": "[concat('ssh ', parameters('adminUsername'), '@', reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn, ' -p 2200')]"
	# 	},
	# 	"Openshift Infra Load Balancer FQDN": {
	# 		"type": "string",
	# 		"value": "[reference(parameters('infraLbPublicIpDnsLabel')).dnsSettings.fqdn]"
	# 	},
	# 	"Node OS Storage Account Name": {
	# 		"type": "string",
	# 		"value": "[variables('newStorageAccountNodeOs')]"
	# 	},
	# 	"Node Data Storage Account Name": {
	# 		"type": "string",
	# 		"value": "[variables('newStorageAccountNodeData')]"
	# 	},
	# 	"Infra Storage Account Name": {
	# 		"type": "string",
	# 		"value": "[variables('newStorageAccountInfra')]"
	# 	}
	# }