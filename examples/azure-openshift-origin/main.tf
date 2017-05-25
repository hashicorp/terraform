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

# ******* VNETS / SUBNETS ***********

resource "azurerm_virtual_network" "vnet" {
  name                = "openshiftvnet"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  address_space       = ["10.0.0.0/8"]
  dns_servers         = ["10.0.0.4", "10.0.0.5"]
}

resource "azurerm_subnet" "master_subnet" {
  name                      = "mastersubnet"
  virtual_network_name      = "${azurerm_virtual_network.vnet.name}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  address_prefix            = "10.1.0.0/16"
  network_security_group_id = "${azurerm_network_security_group.master_nsg.id}"
}

resource "azurerm_subnet" "node_subnet" {
  name                      = "nodesubnet"
  virtual_network_name      = "${azurerm_virtual_network.vnet.name}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  address_prefix            = "10.2.0.0/16"
  network_security_group_id = "${azurerm_network_security_group.node_nsg.id}"
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

# ******* IP ADDRESSES ***********

resource "azurerm_public_ip" "openshift_master_pip" {
  name                         = "masterpip"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  location                     = "${azurerm_resource_group.rg.location}"
  public_ip_address_allocation = "Static"
  domain_name_label            = "${var.openshift_master_public_ip_dns_label}masterpip"
}

resource "azurerm_public_ip" "infra_lb_pip" {
  name                         = "${var.infra_lb_publicip_dns_label}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  location                     = "${azurerm_resource_group.rg.location}"
  public_ip_address_allocation = "Static"
  domain_name_label            = "${var.infra_lb_publicip_dns_label}infrapip"
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

# ******* MASTER LOAD BALANCER ***********

resource "azurerm_lb" "master_lb" {
  name                = "masterloadbalancer"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"

  frontend_ip_configuration {
    name                 = "LoadBalancerFrontEnd"
    public_ip_address_id = "${azurerm_public_ip.openshift_master_pip.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "master_lb" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  name                = "loadBalancerBackEnd"
  loadbalancer_id     = "${azurerm_lb.master_lb.id}"
}

resource "azurerm_lb_probe" "master_lb" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  loadbalancer_id     = "${azurerm_lb.master_lb.id}"
  name                = "8443Probe"
  port                = 8443
  interval_in_seconds = 5
  number_of_probes    = 2
  protocol            = "Tcp"
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
}

# ******* INFRA LOAD BALANCER ***********

resource "azurerm_lb" "infra_lb" {
  name                = "infraloadbalancer"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  location            = "${azurerm_resource_group.rg.location}"

  frontend_ip_configuration {
    name                 = "LoadBalancerFrontEnd"
    public_ip_address_id = "${azurerm_public_ip.infra_lb_pip.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "infra_lb" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  name                = "loadBalancerBackEnd"
  loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
}

resource "azurerm_lb_probe" "infra_lb_http_probe" {
  resource_group_name = "${azurerm_resource_group.rg.name}"
  loadbalancer_id     = "${azurerm_lb.infra_lb.id}"
  name                = "httpProbe"
  port                = 80
  interval_in_seconds = 5
  number_of_probes    = 2
  protocol            = "Tcp"
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
  idle_timeout_in_minutes        = 30
  probe_id                       = "${azurerm_lb_probe.infra_lb_http_probe.id}"
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
  idle_timeout_in_minutes        = 30
  probe_id                       = "${azurerm_lb_probe.infra_lb_https_probe.id}"
}

# ******* NETWORK INTERFACES ***********

resource "azurerm_network_interface" "master_nic" {
  name                      = "masternic${count.index}"
  location                  = "${azurerm_resource_group.rg.location}"
  resource_group_name       = "${azurerm_resource_group.rg.name}"
  network_security_group_id = "${azurerm_network_security_group.master_nsg.id}"
  count                     = "${var.master_instance_count}"

  ip_configuration {
    name                                    = "masteripconfig${count.index}"
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
    name                                    = "infraipconfig${count.index}"
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
    name                                    = "nodeipconfig${count.index}"
    subnet_id                               = "${azurerm_subnet.node_subnet.id}"
    private_ip_address_allocation           = "Dynamic"
    load_balancer_backend_address_pools_ids = ["${azurerm_lb_backend_address_pool.infra_lb.id}"]
  }
}

# ******* Master VMs *******

resource "azurerm_virtual_machine" "master" {
  name                  = "masterVmDeployment${count.index}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.master.id}"
  network_interface_ids = ["${azurerm_network_interface.master_nic.*.id}"]
  vm_size               = "${var.master_vm_size}"
  count                 = "${var.master_instance_count}"

  tags {
    displayName = "${var.openshift_cluster_prefix}-master VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-master"
    admin_username = "${var.admin_username}"
    admin_password = "Password1234!"
  }

  os_profile_linux_config {
    disable_password_authentication = false

    # TODO: ACTUALLY DISABLE PASSWORD AUTHENTICATION
  }

  storage_image_reference {
    publisher = "${lookup(join("_map", var.os_image), "publisher")}"
    offer     = "${lookup(join("_map", var.os_image), "offer")}"
    sku       = "${lookup(join("_map", var.os_image), "sku")}"
    version   = "${lookup(join("_map", var.os_image), "version")}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-master-osdisk"
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}/vhds/${openshift_cluster_prefix}-master-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-master-docker-pool"
    vhd_uri       = "${azurerm_storage_account.master_storage_account.primary_blob_endpoint}/vhds/${openshift_cluster_prefix}-master-docker-pool.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* Infra VMs *******

resource "azurerm_virtual_machine" "infra" {
  name                  = "infraVmDeployment${count.index}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.infra.id}"
  network_interface_ids = ["${azurerm_network_interface.infra_nic.*.id}"]
  vm_size               = "${var.infra_vm_size}"
  count                 = "${var.infra_instance_count}"

  tags {
    displayName = "${var.openshift_cluster_prefix}-infra VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-infra"
    admin_username = "${var.admin_username}"
    admin_password = "Password1234!"
  }

  os_profile_linux_config {
    disable_password_authentication = false

    # TODO: ACTUALLY DISABLE PASSWORD AUTHENTICATION
  }

  storage_image_reference {
    publisher = "${lookup(join("_map", var.os_image), "publisher")}"
    offer     = "${lookup(join("_map", var.os_image), "offer")}"
    sku       = "${lookup(join("_map", var.os_image), "sku")}"
    version   = "${lookup(join("_map", var.os_image), "version")}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-infra-osdisk"
    vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}/vhds/${openshift_cluster_prefix}-infra-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-infra-docker-pool"
    vhd_uri       = "${azurerm_storage_account.infra_storage_account.primary_blob_endpoint}/vhds/${openshift_cluster_prefix}-infra-docker-pool.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# ******* Node VMs *******

resource "azurerm_virtual_machine" "node" {
  name                  = "nodeVmDeployment${count.index}"
  location              = "${azurerm_resource_group.rg.location}"
  resource_group_name   = "${azurerm_resource_group.rg.name}"
  availability_set_id   = "${azurerm_availability_set.master.id}"
  network_interface_ids = ["${azurerm_network_interface.node_nic.*.id}"]
  vm_size               = "${var.node_vm_size}"
  count                 = "${var.node_instance_count}"

  tags {
    displayName = "${var.openshift_cluster_prefix}-node VM Creation"
  }

  os_profile {
    computer_name  = "${var.openshift_cluster_prefix}-node"
    admin_username = "${var.admin_username}"
    admin_password = "Password1234!"
  }

  os_profile_linux_config {
    disable_password_authentication = false

    # TODO: ACTUALLY DISABLE PASSWORD AUTHENTICATION
  }

  storage_image_reference {
    publisher = "${lookup(join("_map", var.os_image), "publisher")}"
    offer     = "${lookup(join("_map", var.os_image), "offer")}"
    sku       = "${lookup(join("_map", var.os_image), "sku")}"
    version   = "${lookup(join("_map", var.os_image), "version")}"
  }

  storage_os_disk {
    name          = "${var.openshift_cluster_prefix}-node-osdisk"
    vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}/vhds/${openshift_cluster_prefix}-node-osdisk.vhd"
    caching       = "ReadWrite"
    create_option = "FromImage"
  }

  storage_data_disk {
    name          = "${var.openshift_cluster_prefix}-node-docker-pool"
    vhd_uri       = "${azurerm_storage_account.nodeos_storage_account.primary_blob_endpoint}/vhds/${openshift_cluster_prefix}-node-docker-pool.vhd"
    disk_size_gb  = "${var.data_disk_size}"
    create_option = "Empty"
    lun           = 0
  }
}

# 		}, {
# 			"type": "Microsoft.Compute/virtualMachines/extensions",
# 			"name": "[concat(variables('openshiftMasterHostname'), '-', copyIndex(), '/deployOpenShift')]",
# 			"location": "[variables('location')]",
# 			"apiVersion": "[variables('apiVersionCompute')]",
# 			"tags": {
# 				"displayName": "PrepMaster"
# 			},
# 			"dependsOn": [
# 				"[concat('masterVmDeployment', copyindex())]"
# 			],
# 			"copy": {
# 				"name": "masterPrepLoop",
# 				"count": "[parameters('masterInstanceCount')]"
# 			},
# 			"properties": {
# 				"publisher": "Microsoft.Azure.Extensions",
# 				"type": "CustomScript",
# 				"typeHandlerVersion": "2.0",
# 				"autoUpgradeMinorVersion": true,
# 				"settings": {
# 					"fileUris": [
# 						"[variables('masterPrepScriptUrl')]"
# 					]
# 				},
# 				"protectedSettings": {
# 					"commandToExecute": "[concat('bash ', variables('masterPrepScriptFileName'), ' ', variables('newStorageAccountPersistentVolume1'), ' ', parameters('adminUsername'))]"
# 				}
# 			}
# 		}, {
# 			"type": "Microsoft.Compute/virtualMachines/extensions",
# 			"name": "[concat(variables('openshiftInfraHostname'), '-', copyIndex(), '/prepNodes')]",
# 			"location": "[variables('location')]",
# 			"apiVersion": "[variables('apiVersionCompute')]",
# 			"tags": {
# 				"displayName": "PrepInfra"
# 			},
# 			"dependsOn": [
# 				"[concat('infraVmDeployment', copyindex())]"
# 			],
# 			"copy": {
# 				"name": "infraPrepLoop",
# 				"count": "[parameters('infraInstanceCount')]"
# 			},
# 			"properties": {
# 				"publisher": "Microsoft.Azure.Extensions",
# 				"type": "CustomScript",
# 				"typeHandlerVersion": "2.0",
# 				"autoUpgradeMinorVersion": true,
# 				"settings": {
# 					"fileUris": [
# 						"[variables('nodePrepScriptUrl')]"
# 					]
# 				},
# 				"protectedSettings": {
# 					"commandToExecute": "[concat('bash ', variables('nodePrepScriptFileName'))]"
# 				}
# 			}
# 		}, {
# 			"type": "Microsoft.Compute/virtualMachines/extensions",
# 			"name": "[concat(variables('openshiftNodeHostname'), '-', copyIndex(), '/prepNodes')]",
# 			"location": "[variables('location')]",
# 			"apiVersion": "[variables('apiVersionCompute')]",
# 			"tags": {
# 				"displayName": "PrepNodes"
# 			},
# 			"dependsOn": [
# 				"[concat('nodeVmDeployment', copyindex())]"
# 			],
# 			"copy": {
# 				"name": "nodePrepLoop",
# 				"count": "[parameters('nodeInstanceCount')]"
# 			},
# 			"properties": {
# 				"publisher": "Microsoft.Azure.Extensions",
# 				"type": "CustomScript",
# 				"typeHandlerVersion": "2.0",
# 				"autoUpgradeMinorVersion": true,
# 				"settings": {
# 					"fileUris": [
# 						"[variables('nodePrepScriptUrl')]"
# 					]
# 				},
# 				"protectedSettings": {
# 					"commandToExecute": "[concat('bash ', variables('nodePrepScriptFileName'))]"
# 				}
# 			}
# 		}, {
# 			"name": "OpenShiftDeployment",
# 			"type": "Microsoft.Resources/deployments",
# 			"apiVersion": "[variables('apiVersionLinkTemplate')]",
# 			"dependsOn": [
# 				"[resourceId('Microsoft.Storage/storageAccounts', variables('newStorageAccountPersistentVolume1'))]",
# 				"[resourceId('Microsoft.Storage/storageAccounts', variables('newStorageAccountRegistry'))]",
# 				"masterPrepLoop",
# 				"infraPrepLoop",
# 				"nodePrepLoop"
# 			],
# 			"properties": {
# 				"mode": "Incremental",
# 				"templateLink": {
# 					"uri": "[variables('openshiftDeploymentTemplateUrl')]",
# 					"contentVersion": "1.0.0.0"
# 				},
# 				"parameters": {
# 					"_artifactsLocation": {
# 						"value": "[parameters('_artifactsLocation')]"
# 					},
# 					"apiVersionCompute": {
# 						"value": "[variables('apiVersionCompute')]"
# 					},
# 					"newStorageAccountRegistry": {
# 						"value": "[variables('newStorageAccountRegistry')]"
# 					},
# 					"newStorageAccountKey": {
# 						"value": "[listKeys(variables('newStorageAccountRegistry'),'2015-06-15').key1]"
# 					},
# 					"newStorageAccountPersistentVolume1": {
# 						"value": "[variables('newStorageAccountPersistentVolume1')]"
# 					},
# 					"newStorageAccountPV1Key": {
# 						"value": "[listKeys(variables('newStorageAccountPersistentVolume1'),'2015-06-15').key1]"
# 					},
# 					"openshiftMasterHostname": {
# 						"value": "[variables('openshiftMasterHostname')]"
# 					},
# 					"openshiftMasterPublicIpFqdn": {
# 						"value": "[reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn]"
# 					},
# 					"openshiftMasterPublicIpAddress": {
# 						"value": "[reference(parameters('openshiftMasterPublicIpDnsLabel')).ipAddress]"
# 					},
# 					"openshiftInfraHostname": {
# 						"value": "[variables('openshiftInfraHostname')]"
# 					},
# 					"openshiftNodeHostname": {
# 						"value": "[variables('openshiftNodeHostname')]"
# 					},
# 					"masterInstanceCount": {
# 						"value": "[parameters('masterInstanceCount')]"
# 					},
# 					"infraInstanceCount": {
# 						"value": "[parameters('infraInstanceCount')]"
# 					},
# 					"nodeInstanceCount": {
# 						"value": "[parameters('nodeInstanceCount')]"
# 					},
# 					"adminUsername": {
# 						"value": "[parameters('adminUsername')]"
# 					},
# 					"openshiftPassword": {
# 						"value": "[parameters('openshiftPassword')]"
# 					},
# 					"aadClientId": {
# 						"value": "[parameters('aadClientId')]"
# 					},
# 					"aadClientSecret": {
# 						"value": "[parameters('aadClientSecret')]"
# 					},
# 					"xipioDomain": {
# 						"value": "[concat(reference(parameters('infraLbPublicIpDnsLabel')).ipAddress, '.xip.io')]"
# 					},
# 					"customDomain": {
# 						"value": "[parameters('defaultSubDomain')]"
# 					},
# 					"subDomainChosen": {
# 						"value": "[concat(parameters('defaultSubDomainType'), 'Domain')]"
# 					},
# 					"sshPrivateKey": {
# 						"reference": {
# 							"keyvault": {
# 								"id": "[concat('/subscriptions/', subscription().subscriptionId, '/resourceGroups/', parameters('keyVaultResourceGroup'), '/providers/Microsoft.KeyVault/vaults/', parameters('keyVaultName'))]"
# 							},
# 							"secretName": "[parameters('keyVaultSecret')]"
# 						}
# 					}
# 				}
# 			}
# 		}
# 	],
# 	"outputs": {
# 		"Openshift Console Url": {
# 			"type": "string",
# 			"value": "[concat('https://', reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn, ':8443/console')]"
# 		},
# 		"Openshift Master SSH": {
# 			"type": "string",
# 			"value": "[concat('ssh ', parameters('adminUsername'), '@', reference(parameters('openshiftMasterPublicIpDnsLabel')).dnsSettings.fqdn, ' -p 2200')]"
# 		},
# 		"Openshift Infra Load Balancer FQDN": {
# 			"type": "string",
# 			"value": "[reference(parameters('infraLbPublicIpDnsLabel')).dnsSettings.fqdn]"
# 		},
# 		"Node OS Storage Account Name": {
# 			"type": "string",
# 			"value": "[variables('newStorageAccountNodeOs')]"
# 		},
# 		"Node Data Storage Account Name": {
# 			"type": "string",
# 			"value": "[variables('newStorageAccountNodeData')]"
# 		},
# 		"Infra Storage Account Name": {
# 			"type": "string",
# 			"value": "[variables('newStorageAccountInfra')]"
# 		}
# 	}
# }

