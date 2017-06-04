variable "location" {}
variable "resource_prefix" {}
variable "webserver_prefix" {}
variable "lb_dns_label" {}

variable "instance_count" {}
variable "instance_vmprofile" {}

variable "image_admin_username" {}
variable "image_admin_password" {}

variable "image_publisher" {}
variable "image_offer" {}
variable "image_sku" {}
variable "image_version" {}

# Create webserver resource group
resource "azurerm_resource_group" "webservers_rg" {
  name     = "${var.resource_prefix}_rg"
  location = "${var.location}"
}

# Create virtual network
resource "azurerm_virtual_network" "webservers_vnet" {
  name                = "webservers_vnet"
  address_space       = ["10.1.0.0/24"]
  location = "${var.location}"
  resource_group_name = "${azurerm_resource_group.webservers_rg.name}"
}

# Create subnet
resource "azurerm_subnet" "webservers_subnet" {
  name                 = "webservers_subnet"
  resource_group_name  = "${azurerm_resource_group.webservers_rg.name}"
  virtual_network_name = "${azurerm_virtual_network.webservers_vnet.name}"
  address_prefix       = "10.1.0.0/24"
}

# Create a public ip for the location LB
resource "azurerm_public_ip" "webserverpublic_ip" {
  name                          = "${var.resource_prefix}_publicip"
  location                      = "${var.location}"
  resource_group_name           = "${azurerm_resource_group.webservers_rg.name}"
  public_ip_address_allocation  = "static"
  domain_name_label             = "${var.lb_dns_label}"
}

# Create webservers LB
resource "azurerm_lb" "webservers_lb" {
  name                = "webservers_lb"
  location            = "${var.location}"
  resource_group_name = "${azurerm_resource_group.webservers_rg.name}"

  frontend_ip_configuration {
    name                 = "webserverpublic_ip"
    public_ip_address_id = "${azurerm_public_ip.webserverpublic_ip.id}"
  }
}

# Add the backend for webserver LB
resource "azurerm_lb_backend_address_pool" "webservers_lb_backend" {
  name                = "webservers_lb_backend"
  resource_group_name = "${azurerm_resource_group.webservers_rg.name}"
  loadbalancer_id     = "${azurerm_lb.webservers_lb.id}"
}

# Create HTTP probe on port 80
resource "azurerm_lb_probe" "httpprobe" {
  name                = "httpprobe"
  resource_group_name = "${azurerm_resource_group.webservers_rg.name}"
  loadbalancer_id     = "${azurerm_lb.webservers_lb.id}"
  protocol            = "tcp"  
  port                = 80
}

# Create LB rule for HTTP and add to webserver LB
resource "azurerm_lb_rule" "webservers_lb_http" {
  name                           = "webservers_lb_http"
  resource_group_name            = "${azurerm_resource_group.webservers_rg.name}"
  loadbalancer_id                = "${azurerm_lb.webservers_lb.id}"
  protocol                       = "Tcp"
  frontend_port                  = "80"
  backend_port                   = "80"
  frontend_ip_configuration_name = "webserverpublic_ip"
  probe_id                       = "${azurerm_lb_probe.httpprobe.id}"  
  backend_address_pool_id        = "${azurerm_lb_backend_address_pool.webservers_lb_backend.id}"
}

# Create storage account
resource "azurerm_storage_account" "webservers_sa" {
  name                =  "${var.resource_prefix}storage"
  resource_group_name = "${azurerm_resource_group.webservers_rg.name}"
  location            = "${var.location}"
  account_type        = "Standard_LRS"
}

# Create container
resource "azurerm_storage_container" "webservers_ct" {
  name                  = "vhds"
  resource_group_name   = "${azurerm_resource_group.webservers_rg.name}"
  storage_account_name  = "${azurerm_storage_account.webservers_sa.name}"
  container_access_type = "private"
}

# Configure the scale set using library image
resource "azurerm_virtual_machine_scale_set" "webserver_ss" {
  name                 = "webserver_ss"
  location             = "${var.location}"
  resource_group_name  = "${azurerm_resource_group.webservers_rg.name}"
  upgrade_policy_mode  = "Manual"

  sku {
    name     = "${var.instance_vmprofile}"
    tier     = "Standard"
    capacity = "${var.instance_count}"
  }

  os_profile {
    computer_name_prefix = "${var.webserver_prefix}"
    admin_username       = "${var.image_admin_username}"
    admin_password       = "${var.image_admin_password}"
  }

  os_profile_linux_config {
    disable_password_authentication = false
  }

  network_profile {
    name    = "web_ss_net_profile"
    primary = true

    ip_configuration {
      name                                   = "web_ss_ip_profile"
      subnet_id                              = "${azurerm_subnet.webservers_subnet.id}"
      load_balancer_backend_address_pool_ids = ["${azurerm_lb_backend_address_pool.webservers_lb_backend.id}"]
    }
  }

  storage_profile_os_disk {
    name           = "osDiskProfile"
    caching        = "ReadWrite"
    create_option  = "FromImage"
    vhd_containers = ["${azurerm_storage_account.webservers_sa.primary_blob_endpoint}${azurerm_storage_container.webservers_ct.name}"]
  }

  storage_profile_image_reference {
    publisher = "${var.image_publisher}"
    offer     = "${var.image_offer}"
    sku       = "${var.image_sku}"
    version   = "${var.image_version}"
  }

  extension {
    name = "CustomScriptForLinux"
    publisher = "Microsoft.OSTCExtensions"
    type = "CustomScriptForLinux"
    type_handler_version = "1.4"
    settings = <<SETTINGS
    {
      "commandToExecute" : "sudo apt-get -y install apache2"
    }
    SETTINGS
  }

}