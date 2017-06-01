# Traffic manager settings

variable "global_location" {
  default        = "UK West"
  description    = "Where any global resources will be placed"
}

variable "dns_relative_name" {
  default        = "azuretfexample"
  description    = "Relative DNS name for traffic manager"
}

# Location 01 Settings

variable "location01_location" {
  default        = "UK West"
  description    = "First location to build"
}

variable "location01_resource_prefix" {
  default        = "ukwestweb"
  description    = "Prefix for naming resource group"
}

variable "location01_webserver_prefix" {
  default        = "ukwwebsvr"
  description    = "Prefix for naming web servers"
}

variable "location01_lb_dns_label" {
  default        = "ukwestwebexample"
  description    = "DNS name label for the locations load balancer"
}

# Location 02 Settings

variable "location02_location" {
  default        = "West US"
  description    = "Second location to build"
}

variable "location02_resource_prefix" {
  default        = "uswestweb"
  description    = "Prefix for naming resource group"
}

variable "location02_webserver_prefix" {
  default        = "uswwebsvr"
  description    = "Prefix for naming web servers"
}

variable "location02_lb_dns_label" {
  default        = "uswestwebexample"
  description    = "DNS name label for the locations load balancer"
}

# Scale set and VM settings

variable "instance_count" {
  default        = "2"
  description    = "Number of server instances to create in scale set"
}

variable "instance_vmprofile" {
  default        = "Standard_A1"
  description    = "VM profile of servers in scale set"
}

# OS Profile

variable "image_admin_username" {
  default        = "webadmin"
  description    = "Local admin user name"
}

variable "image_admin_password" {
  default        = "2nmn39x#3775hh3x9"
  description    = "Password"
}

# Market place image to use

variable "image_publisher" {
  default        = "Canonical"
  description    = "Publisher of market place image"
}
variable "image_offer" {
  default        = "UbuntuServer"
  description    = "Market place image name"
}
variable "image_sku" {
  default        = "16.10"
  description    = "Market place image SKU"
}
variable "image_version" {
  default        = "latest"
  description    = "Market place image version"
}
