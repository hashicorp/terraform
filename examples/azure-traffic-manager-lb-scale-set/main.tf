# Provider accounts must be passed 

variable "subscription_id" {}
variable "client_id" {}
variable "client_secret" {}
variable "tenant_id" {}

provider "azurerm" {
  subscription_id = "${var.subscription_id}"
  client_id       = "${var.client_id}"
  client_secret   = "${var.client_secret}"
  tenant_id       = "${var.tenant_id}"
  }

# Create the resource group and assets for first location
module "location01" {
  source                = "./tf_modules"

  location              = "${var.location01_location}"
  resource_prefix       = "${var.location01_resource_prefix}"
  webserver_prefix      = "${var.location01_webserver_prefix}"
  lb_dns_label          = "${var.location01_lb_dns_label}"  
  
  instance_count        = "${var.instance_count}"
  instance_vmprofile    = "${var.instance_vmprofile}"

  image_admin_username  = "${var.image_admin_username}"
  image_admin_password  = "${var.image_admin_password}"

  image_publisher       = "${var.image_publisher}"
  image_offer           = "${var.image_offer}"
  image_sku             = "${var.image_sku}"
  image_version         = "${var.image_version}"
  
}

# Create the resource group and assets for second location
module "location02" {
  source             = "./tf_modules"

  location           = "${var.location02_location}"
  resource_prefix    = "${var.location02_resource_prefix}"
  webserver_prefix   = "${var.location02_webserver_prefix}"
  lb_dns_label       = "${var.location02_lb_dns_label}"  

  instance_count     = "${var.instance_count}"
  instance_vmprofile = "${var.instance_vmprofile}"

  image_admin_username  = "${var.image_admin_username}"
  image_admin_password  = "${var.image_admin_password}"

  image_publisher       = "${var.image_publisher}"
  image_offer           = "${var.image_offer}"
  image_sku             = "${var.image_sku}"
  image_version         = "${var.image_version}"

}

# Create global resource group
resource "azurerm_resource_group" "global_rg" {
  name     = "global_rg"
  location = "${var.global_location}"
}

# Create the traffic manager
resource "azurerm_traffic_manager_profile" "trafficmanagerhttp" {
  name                = "trafficmanagerhttp"
  resource_group_name = "${azurerm_resource_group.global_rg.name}"

  traffic_routing_method = "Weighted"

  dns_config {
    relative_name = "${var.dns_relative_name}"
    ttl           = 100
  }

  monitor_config {
    protocol = "http"
    port     = 80
    path     = "/"
  }
}

# Add endpoint mappings to traffic manager, location01
resource "azurerm_traffic_manager_endpoint" "trafficmanagerhttp_01" {
  name                = "trafficmanagerhttp_ukw"
  resource_group_name = "${azurerm_resource_group.global_rg.name}"
  profile_name        = "${azurerm_traffic_manager_profile.trafficmanagerhttp.name}"
  target_resource_id  = "${module.location01.webserverpublic_ip_id}"
  type                = "azureEndpoints"
  weight              = 100
}

# Add endpoint mappings to traffic manager, location02
resource "azurerm_traffic_manager_endpoint" "trafficmanagerhttp_02" {
  name                = "trafficmanagerhttp_wus"
  resource_group_name = "${azurerm_resource_group.global_rg.name}"
  profile_name        = "${azurerm_traffic_manager_profile.trafficmanagerhttp.name}"
  target_resource_id  = "${module.location02.webserverpublic_ip_id}"
  type                = "azureEndpoints"
  weight              = 100
}