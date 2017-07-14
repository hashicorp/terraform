# variable "client_id" {}
# variable "client_secret" {}
# variable "tenant_id" {}
# variable "subscription_id" {}

variable "resource_group" {
  description = "Resource group name into which your Spark and Cassandra deployment will go."
}

variable "location" {
  description = "The location/region where the virtual network is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "unique_prefix" {
  description = "This prefix is used for names which need to be globally unique."
}

variable "vm_admin_username" {
  description = "Specify an admin username that should be used to login to the VM. Min length: 1"
}

variable "vm_admin_password" {
  description = "Specify an admin password that should be used to login to the VM. Must be between 6-72 characters long and must satisfy at least 3 of password complexity requirements from the following: 1) Contains an uppercase character 2) Contains a lowercase character 3) Contains a numeric digit 4) Contains a special character"
}

variable "os_image_publisher" {
  description = "name of the publisher of the image (az vm image list)"
  default     = "OpenLogic"
}

variable "os_image_offer" {
  description = "the name of the offer (az vm image list)"
  default     = "CentOS"
}

variable "os_version" {
  description = "version of the image to apply (az vm image list)"
  default     = "6.5"
}

variable "api_version" {
  default = "2015-06-15"
}

variable "artifacts_location" {
  description = "The base URI where artifacts required by this template are located."
  default     = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/wordpress-mysql-replication/"
}

variable "azuremysql_script" {
  description = "The directory and script which will configure MySQL"
  default     = "scripts/azuremysql.sh"
}

variable "mysql_cfg_file_path" {
  description = "The directory and script which will be called in the extension for the MySQL config"
  default     = "scripts/my.cnf.template"
}

variable "site_name" {
  description = "wordpress site name"
}

variable "hosting_plan_name" {
  description = "website host plan"
}

variable "sku" {
  description = "Website sku. Allowed values: Basic, Standard, Premium"
  default     = "Standard"
}

variable "worker_size" {
  description = "Website worker size. Allowed values: 0, 1, 2"
  default     = "1"
}

variable "dns_name" {
  description = "Connect to your cluster using dnsName.location.cloudapp.azure.com"
}

variable "public_ip_name" {
  description = "public IP name for MySQL loadbalancer"
  default     = "mysqlIP01"
}

variable "mysql_root_password" {
  description = "mysql root user password"
}

variable "mysql_replication_password" {
  description = "mysql replication user password"
}

variable "mysql_probe_password" {
  description = "mysql probe password"
}

variable "vm_size" {
  description = "size for the VMs"
  default     = "Standard_D2"
}

variable "storage_account_type" {
  description = "Storage account type for the cluster"
  default     = "Standard_LRS"
}

variable "virtual_network_name" {
  description = "New or Existing Virtual network name for the cluster"
  default     = "mysqlvnet"
}

variable "vnet_new_or_existing" {
  description = "Identifies whether to use new or existing Virtual Network"
  default     = "new"
}

variable "vnet_existing_resource_group_name" {
  description = "If using existing VNet, specifies the resource group for the existing VNet"
  default     = ""
}

variable "db_subnet_name" {
  description = "subnet name for the MySQL nodes"
  default     = "default"
}

variable "vnet_address_prefix" {
  description = "IP address in CIDR for virtual network"
  default     = "10.0.0.0/16"
}

variable "db_subnet_address_prefix" {
  description = "IP address in CIDR for db subnet"
  default     = "10.0.1.0/24"
}

variable "db_subnet_start_address" {
  description = "Start IP address for the VMs in db subnet"
  default     = "10.0.1.4"
}

variable "image_publisher" {
  description = "publisher for the VM OS image"
  default     = "OpenLogic"
}

variable "image_offer" {
  description = "VM OS name"
  default     = "CentOS"
}

variable "image_sku" {
  description = "VM OS version. Allowed values: 6.5, 6.6"
  default     = "6.5"
}

variable "mysql_front_end_port_0" {
  description = "MySQL public port"
  default     = "3306"
}

variable "mysql_front_end_port_1" {
  description = "MySQL public port"
  default     = "3307"
}

variable "ssh_nat_rule_front_end_port_0" {
  description = "public ssh port for VM1"
  default     = "64001"
}

variable "ssh_nat_rule_front_end_port_1" {
  description = "public ssh port for VM2"
  default     = "64002"
}

variable "mysql_probe_port_0" {
  description = "MySQL public port master"
  default     = "9200"
}

variable "mysql_probe_port_1" {
  description = "MySQL public port slave"
  default     = "9201"
}

variable "storage_account_name" {
  description = "Name of the Storage Account"
  default     = "storagesa"
}

variable "template_api_version" {
  default = "2015-01-01"
}

variable "wpdbname" {
  default = "wordpress"
}

variable "node_count" {
  default = 2
}

variable "nic_name" {
  description = "Name of the Network Interface"
  default     = "nic"
}
