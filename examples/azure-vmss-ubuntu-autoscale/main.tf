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
  name                = "${var.resource_group}vnet"
  location            = "${var.location}"
  address_space       = ["10.0.0.0/16"]
  resource_group_name = "${azurerm_resource_group.rg.name}"

  subnet {
    name           = "subnet1"
    address_prefix = "10.0.0.0/24"
  }
}

resource "azurerm_subnet" "subnet1" {
  name                = "subnet1"
  location            = "${var.location}"
  address_prefix      = "10.0.0.0/24"
  resource_group_name = "${azurerm_resource_group.rg.name}"
}

resource "azurerm_public_ip" "pip" {
  name                         = "PublicIp1"
  location                     = "${var.location}"
  resource_group_name          = "${azurerm_resource_group.rg.name}"
  public_ip_address_allocation = "Dynamic"
  domain_name_label            = "${var.vmss_name}"
}

resource "azurerm_network_interface" "nic" {
  name                = "${var.rg_prefix}nic"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  ip_configuration {
    name                          = "${var.rg_prefix}ipconfig"
    subnet_id                     = "${azurerm_subnet.subnet.id}"
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = "${azurerm_public_ip.pip.id}"
  }
}

resource "azurerm_lb" "lb" {
  name                = "LoadBalancer"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"

  frontend_ip_configuration {
    name                 = "LBFrontEnd"
    public_ip_address_id = "${azurerm_public_ip.pip.id}"
    subnet_id            = "${azurerm_subnet.subnet.id}"
  }
}

resource "azurerm_lb_backend_address_pool" "backlb" {
  name                = "BackEndAddressPool"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  loadbalancer_id     = "${azurerm_lb.lb.id}"
}

resource "azurerm_lb_nat_pool" "np" {
  resource_group_name            = "${azurerm_resource_group.rg.name}"
  loadbalancer_id                = "${azurerm_lb.lb.id}"
  name                           = "NATPool"
  protocol                       = "Tcp"
  frontend_port_start            = 50000
  frontend_port_end              = 50119
  backend_port                   = 22
  frontend_ip_configuration_name = "LBFrontEnd"
}

resource "azurerm_storage_account" "stor" {
  name                = "${var.resource_group}stor"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  account_type        = "${var.storage_account_type}"
}

resource "azurerm_virtual_machine_scale_set" "scaleset" {
  name                = "autoscalewad"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.rg.name}"
  upgrade_policy_mode = "Manual"
  overprovision       = true

  sku {
    name     = "${var.vm_sku}"
    tier     = "Standard"
    capacity = "${var.instance_count}"
  }

  os_profile {
    computer_name_prefix = "${var.vmss_name}"
    admin_username       = "${var.admin_username}"
    admin_password       = "${var.admin_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }

  network_profile {
    name    = "${azurerm_network_interface.nic.name}"
    primary = true

    ip_configuration {
      name                                   = "IPConfiguration"
      subnet_id                              = "${azurerm_subnet.subnet1.id}"
      load_balancer_backend_address_pool_ids = ["${azurerm_lb_backend_address_pool.backlb.id}"]
      load_balancer_inbound_nat_pool_ids     = ["${azurerm_lb_nat_pool.np.id}"]
    }
  }

  storage_profile_os_disk {
    name           = "osDiskProfile"
    caching        = "ReadWrite"
    create_option  = "FromImage"
    vhd_containers = ["${azurerm_storage_account.stor.primary_blob_endpoint}${azurerm_storage_container.stor.name}"]
  }

  storage_profile_image_reference {
    publisher = "${var.image_publisher}"
    offer     = "${var.image_offer}"
    sku       = "${var.ubuntu_os_version}"
    version   = "latest"
  }
}

# "type": "Microsoft.Insights/autoscaleSettings",
# "apiVersion": "[variables('insightsApiVersion')]",
# "name": "autoscalewad",
# "location": "[resourceGroup().location]",
# "dependsOn": [
#   "[concat('Microsoft.Compute/virtualMachineScaleSets/', variables('namingInfix'))]"
# ],
# "properties": {
#   "name": "autoscalewad",
#   "targetResourceUri": "[concat('/subscriptions/',subscription().subscriptionId, '/resourceGroups/',  resourceGroup().name, '/providers/Microsoft.Compute/virtualMachineScaleSets/', variables('namingInfix'))]",
#   "enabled": true,
#   "profiles": [
#     {
#       "name": "Profile1",
#       "capacity": {
#         "minimum": "1",
#         "maximum": "10",
#         "default": "1"
#       },
#       "rules": [
#         {
#           "metricTrigger": {
#             "metricName": "Percentage CPU",
#             "metricNamespace": "",
#             "metricResourceUri": "[concat('/subscriptions/',subscription().subscriptionId, '/resourceGroups/',  resourceGroup().name, '/providers/Microsoft.Compute/virtualMachineScaleSets/', variables('namingInfix'))]",
#             "timeGrain": "PT1M",
#             "statistic": "Average",
#             "timeWindow": "PT5M",
#             "timeAggregation": "Average",
#             "operator": "GreaterThan",
#             "threshold": 60.0
#           },
#           "scaleAction": {
#             "direction": "Increase",
#             "type": "ChangeCount",
#             "value": "1",
#             "cooldown": "PT1M"
#           }
#         },
#         {
#           "metricTrigger": {
#             "metricName": "Percentage CPU",
#             "metricNamespace": "",
#             "metricResourceUri": "[concat('/subscriptions/',subscription().subscriptionId, '/resourceGroups/',  resourceGroup().name, '/providers/Microsoft.Compute/virtualMachineScaleSets/', variables('namingInfix'))]",
#             "timeGrain": "PT1M",
#             "statistic": "Average",
#             "timeWindow": "PT5M",
#             "timeAggregation": "Average",
#             "operator": "LessThan",
#             "threshold": 30.0
#           },
#           "scaleAction": {
#             "direction": "Decrease",
#             "type": "ChangeCount",
#             "value": "1",
#             "cooldown": "PT5M"
#           }

