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

resource "azurerm_virtual_network" "vnet" {
  name                = "openshiftvnet"
  location            = "${azurerm_resource_group.rg.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  address_space       = ["10.0.0.0/8"]
  dns_servers         = ["10.0.0.4", "10.0.0.5"]

  subnet {
    name           = "mastersubnet"
    address_prefix = "10.1.0.0/16"
    security_group = "${azurerm_network_security_group.master_nsg.id}"
  }

  subnet {
    name           = "nodesubnet"
    address_prefix = "10.2.0.0/16"
    security_group = "${azurerm_network_security_group.node_nsg.id}"
  }
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

# 			"type": "Microsoft.Network/networkInterfaces",
# 			"name": "[concat(variables('openshiftMasterHostname'), '-', copyIndex(), '-nic')]",
# 			"location": "[variables('location')]",
# 			"apiVersion": "[variables('apiVersionNetwork')]",
# 			"tags": {
# 				"displayName": "OpenShiftMasterNetworkInterface"
# 			},
# 			"dependsOn": [
# 				"[concat('Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'))]",
# 				"[concat('Microsoft.Network/loadBalancers/', variables('masterLoadBalancerName'))]",
# 				"[concat(variables('masterLbId'), '/inboundNatRules/SSH-', variables('openshiftMasterHostname') ,copyIndex())]",
# 				"[concat('Microsoft.Network/networkSecurityGroups/', variables('openshiftMasterHostname'), '-nsg')]"
# 			],
# 			"copy": {
# 				"name": "masterNicLoop",
# 				"count": "[parameters('masterInstanceCount')]"
# 			},
# 			"properties": {
# 				"ipConfigurations": [{
# 					"name": "[concat(variables('openshiftMasterHostname'), copyIndex(), 'ipconfig')]",
# 					"properties": {
# 						"privateIPAllocationMethod": "Dynamic",
# 						"subnet": {
# 							"id": "[concat('/subscriptions/', subscription().subscriptionId, '/resourceGroups/', resourceGroup().name, '/providers/Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'), '/subnets/', variables('masterSubnetName'))]"
# 						},
# 						"loadBalancerBackendAddressPools": [{
# 							"id": "[concat('/subscriptions/', subscription().subscriptionId, '/resourceGroups/', resourceGroup().name, '/providers/Microsoft.Network/loadBalancers/', variables('masterLoadBalancerName'), '/backendAddressPools/loadBalancerBackEnd')]"
# 						}],
# 						"loadBalancerInboundNatRules": [
# 							{
# 								"id": "[concat(variables('masterLbId'),'/inboundNatRules/SSH-', variables('openshiftMasterHostname'), copyIndex())]"
# 							}
# 						]
# 					}
# 				}],
# 				"networkSecurityGroup": {
# 					"id": "[resourceId('Microsoft.Network/networkSecurityGroups', concat(variables('openshiftMasterHostname'), '-nsg'))]"
# 				}
# 			}
# 		}, {
# 			"type": "Microsoft.Network/networkInterfaces",
# 			"name": "[concat(variables('openshiftInfraHostname'), '-', copyIndex(), '-nic')]",
# 			"location": "[variables('location')]",
# 			"apiVersion": "[variables('apiVersionNetwork')]",
# 			"tags": {
# 				"displayName": "OpenShiftInfraNetworkInterfaces"
# 			},
# 			"dependsOn": [
# 				"[concat('Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'))]",
# 				"[concat('Microsoft.Network/loadBalancers/', variables('infraLoadBalancerName'))]",
# 				"[concat('Microsoft.Network/networkSecurityGroups/', variables('openshiftInfraHostname'), '-nsg')]"
# 			],
# 			"copy": {
# 				"name": "infraNicLoop",
# 				"count": "[parameters('infraInstanceCount')]"
# 			},
# 			"properties": {
# 				"ipConfigurations": [{
# 					"name": "[concat(variables('openshiftInfraHostname'), copyIndex(), 'ipconfig')]",
# 					"properties": {
# 						"privateIPAllocationMethod": "Dynamic",
# 						"subnet": {
# 							"id": "[concat('/subscriptions/', subscription().subscriptionId, '/resourceGroups/', resourceGroup().name, '/providers/Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'), '/subnets/', variables('masterSubnetName'))]"
# 						},
# 						"loadBalancerBackendAddressPools": [{
# 							"id": "[concat('/subscriptions/', subscription().subscriptionId, '/resourceGroups/', resourceGroup().name, '/providers/Microsoft.Network/loadBalancers/', variables('infraLoadBalancerName'), '/backendAddressPools/loadBalancerBackEnd')]"
# 						}]
# 					}
# 				}],
# 				"networkSecurityGroup": {
# 					"id": "[resourceId('Microsoft.Network/networkSecurityGroups', concat(variables('openshiftInfraHostname'), '-nsg'))]"
# 				}
# 			}
# 		}, {
# 			"type": "Microsoft.Network/networkInterfaces",
# 			"name": "[concat(variables('openshiftNodeHostname'), '-', copyIndex(), '-nic')]",
# 			"location": "[variables('location')]",
# 			"apiVersion": "[variables('apiVersionNetwork')]",
# 			"tags": {
# 				"displayName": "OpenShiftNodeNetworkInterfaces"
# 			},
# 			"dependsOn": [
# 				"[concat('Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'))]",
# 				"[concat('Microsoft.Network/networkSecurityGroups/', variables('openshiftNodeHostname'), '-nsg')]"
# 			],
# 			"copy": {
# 				"name": "nodeNicLoop",
# 				"count": "[parameters('nodeInstanceCount')]"
# 			},
# 			"properties": {
# 				"ipConfigurations": [{
# 					"name": "[concat(variables('openshiftNodeHostname'), copyIndex(), 'ipconfig')]",
# 					"properties": {
# 						"privateIPAllocationMethod": "Dynamic",
# 						"subnet": {
# 							"id": "[concat('/subscriptions/', subscription().subscriptionId, '/resourceGroups/', resourceGroup().name, '/providers/Microsoft.Network/virtualNetworks/', variables('virtualNetworkName'), '/subnets/', variables('nodeSubnetName'))]"
# 						}
# 					}
# 				}],
# 				"networkSecurityGroup": {
# 					"id": "[resourceId('Microsoft.Network/networkSecurityGroups', concat(variables('openshiftNodeHostname'), '-nsg'))]"
# 				}
# 			}
# 		}, {
# 			"name": "[concat('masterVmDeployment', copyindex())]",
# 			"type": "Microsoft.Resources/deployments",
# 			"apiVersion": "[variables('apiVersionLinkTemplate')]",
# 			"dependsOn": [
# 				"[resourceId('Microsoft.Storage/storageAccounts', variables('newStorageAccountMaster'))]",
# 				"masterNicLoop",
# 				"masteravailabilityset"
# 			],
# 			"copy": {
# 				"name": "masterVmLoop",
# 				"count": "[parameters('masterInstanceCount')]"
# 			},
# 			"properties": {
# 				"mode": "Incremental",
# 				"templateLink": {
# 					"uri": "[variables('clusterNodeDeploymentTemplateUrl')]",
# 					"contentVersion": "1.0.0.0"
# 				},
# 				"parameters": {
# 					"location": {
# 						"value": "[variables('location')]"
# 					},
# 					"sshKeyPath": {
# 						"value": "[variables('sshKeyPath')]"
# 					},
# 					"sshPublicKey": {
# 						"value": "[parameters('sshPublicKey')]"
# 					},
# 					"dataDiskSize": {
# 						"value": "[parameters('dataDiskSize')]"
# 					},
# 					"adminUsername": {
# 						"value": "[parameters('adminUsername')]"
# 					},
# 					"vmSize": {
# 						"value": "[parameters('masterVmSize')]"
# 					},
# 					"availabilitySet": {
# 						"value": "masteravailabilityset"
# 					},
# 					"osImage": {
# 						"value": "[parameters('osImage')]"
# 					},
# 					"hostname": {
# 						"value": "[concat(variables('openshiftMasterHostname'), '-', copyIndex())]"
# 					},
# 					"newStorageAccountOs": {
# 						"value": "[variables('newStorageAccountMaster')]"
# 					},
# 					"newStorageAccountData": {
# 						"value": "[variables('newStorageAccountMaster')]"
# 					},
# 					"apiVersionStorage": {
# 						"value": "[variables('apiVersionStorage')]"
# 					},
# 					"apiVersionCompute": {
# 						"value": "[variables('apiVersionCompute')]"
# 					}
# 				}
# 			}
# 		}, {
# 			"name": "[concat('infraVmDeployment', copyindex())]",
# 			"type": "Microsoft.Resources/deployments",
# 			"apiVersion": "[variables('apiVersionLinkTemplate')]",
# 			"dependsOn": [
# 				"[resourceId('Microsoft.Storage/storageAccounts', variables('newStorageAccountInfra'))]",
# 				"infraNicLoop",
# 				"infraavailabilityset"
# 			],
# 			"copy": {
# 				"name": "infraVmLoop",
# 				"count": "[parameters('infraInstanceCount')]"
# 			},
# 			"properties": {
# 				"mode": "Incremental",
# 				"templateLink": {
# 					"uri": "[variables('clusterNodeDeploymentTemplateUrl')]",
# 					"contentVersion": "1.0.0.0"
# 				},
# 				"parameters": {
# 					"location": {
# 						"value": "[variables('location')]"
# 					},
# 					"sshKeyPath": {
# 						"value": "[variables('sshKeyPath')]"
# 					},
# 					"sshPublicKey": {
# 						"value": "[parameters('sshPublicKey')]"
# 					},
# 					"dataDiskSize": {
# 						"value": "[parameters('dataDiskSize')]"
# 					},
# 					"adminUsername": {
# 						"value": "[parameters('adminUsername')]"
# 					},
# 					"vmSize": {
# 						"value": "[parameters('infraVmSize')]"
# 					},
# 					"availabilitySet": {
# 						"value": "infraavailabilityset"
# 					},
# 					"osImage": {
# 						"value": "[parameters('osImage')]"
# 					},
# 					"hostname": {
# 						"value": "[concat(variables('openshiftInfraHostname'), '-', copyIndex())]"
# 					},
# 					"newStorageAccountOs": {
# 						"value": "[variables('newStorageAccountInfra')]"
# 					},
# 					"newStorageAccountData": {
# 						"value": "[variables('newStorageAccountInfra')]"
# 					},
# 					"apiVersionStorage": {
# 						"value": "[variables('apiVersionStorage')]"
# 					},
# 					"apiVersionCompute": {
# 						"value": "[variables('apiVersionCompute')]"
# 					}
# 				}
# 			}
# 		}, {
# 			"name": "[concat('nodeVmDeployment', copyindex())]",
# 			"type": "Microsoft.Resources/deployments",
# 			"apiVersion": "[variables('apiVersionLinkTemplate')]",
# 			"dependsOn": [
# 				"[resourceId('Microsoft.Storage/storageAccounts', variables('newStorageAccountNodeOs'))]",
# 				"[resourceId('Microsoft.Storage/storageAccounts', variables('newStorageAccountNodeData'))]",
# 				"nodeNicLoop",
# 				"nodeavailabilityset"
# 			],
# 			"copy": {
# 				"name": "nodeVmLoop",
# 				"count": "[parameters('nodeInstanceCount')]"
# 			},
# 			"properties": {
# 				"mode": "Incremental",
# 				"templateLink": {
# 					"uri": "[variables('clusterNodeDeploymentTemplateUrl')]",
# 					"contentVersion": "1.0.0.0"
# 				},
# 				"parameters": {
# 					"location": {
# 						"value": "[variables('location')]"
# 					},
# 					"sshKeyPath": {
# 						"value": "[variables('sshKeyPath')]"
# 					},
# 					"sshPublicKey": {
# 						"value": "[parameters('sshPublicKey')]"
# 					},
# 					"dataDiskSize": {
# 						"value": "[parameters('dataDiskSize')]"
# 					},
# 					"adminUsername": {
# 						"value": "[parameters('adminUsername')]"
# 					},
# 					"vmSize": {
# 						"value": "[parameters('nodeVmSize')]"
# 					},
# 					"availabilitySet": {
# 						"value": "nodeavailabilityset"
# 					},
# 					"osImage": {
# 						"value": "[parameters('osImage')]"
# 					},
# 					"hostname": {
# 						"value": "[concat(variables('openshiftNodeHostname'), '-', copyIndex())]"
# 					},
# 					"newStorageAccountOs": {
# 						"value": "[variables('newStorageAccountNodeOs')]"
# 					},
# 					"newStorageAccountData": {
# 						"value": "[variables('newStorageAccountNodeData')]"
# 					},
# 					"apiVersionStorage": {
# 						"value": "[variables('apiVersionStorage')]"
# 					},
# 					"apiVersionCompute": {
# 						"value": "[variables('apiVersionCompute')]"
# 					}
# 				}
# 			}
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