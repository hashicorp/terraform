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

variable "storage_master_type" {
  description = "Storage type that is used for master Spark node.  This storage account is used to store VM disks. Allowed values: Standard_LRS, Standard_ZRS, Standard_GRS, Standard_RAGRS, Premium_LRS"
  default     = "Standard_LRS"
}

variable "storage_slave_type" {
  description = "Storage type that is used for each of the slave Spark node.  This storage account is used to store VM disks. Allowed values : Standard_LRS, Standard_ZRS, Standard_GRS, Standard_RAGRS, Premium_LRS"
  default     = "Standard_LRS"
}

variable "storage_cassandra_type" {
  description = "Storage type that is used for Cassandra.  This storage account is used to store VM disks. Allowed values: Standard_LRS, Standard_ZRS, Standard_GRS, Standard_RAGRS, Premium_LRS"
  default     = "Standard_LRS"
}

variable "vm_master_vm_size" {
  description = "VM size for master Spark node.  This VM can be sized smaller. Allowed values: Standard_D1_v2, Standard_D2_v2, Standard_D3_v2, Standard_D4_v2, Standard_D5_v2, Standard_D11_v2, Standard_D12_v2, Standard_D13_v2, Standard_D14_v2, Standard_A8, Standard_A9, Standard_A10, Standard_A11"
  default     = "Standard_D1_v2"
}

variable "vm_number_of_slaves" {
  description = "Number of VMs to create to support the slaves.  Each slave is created on it's own VM.  Minimum of 2 & Maximum of 200 VMs. min = 2, max = 200"
  default     = 2
}

variable "vm_slave_vm_size" {
  description = "VM size for slave Spark nodes.  This VM should be sized based on workloads. Allowed values: Standard_D1_v2, Standard_D2_v2, Standard_D3_v2, Standard_D4_v2, Standard_D5_v2, Standard_D11_v2, Standard_D12_v2, Standard_D13_v2, Standard_D14_v2, Standard_A8, Standard_A9, Standard_A10, Standard_A11"
  default     = "Standard_D3_v2"
}

variable "vm_cassandra_vm_size" {
  description = "VM size for Cassandra node.  This VM should be sized based on workloads. Allowed values: Standard_D1_v2, Standard_D2_v2, Standard_D3_v2, Standard_D4_v2, Standard_D5_v2, Standard_D11_v2, Standard_D12_v2, Standard_D13_v2, Standard_D14_v2, Standard_A8, Standard_A9, Standard_A10, Standard_A11"
  default     = "Standard_D3_v2"
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
  default     = "7.3"
}

variable "api_version" {
  default = "2015-06-15"
}

variable "artifacts_location" {
  description = "The base URI where artifacts required by this template are located."
  default     = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/spark-and-cassandra-on-centos/CustomScripts/"
}

variable "vnet_spark_prefix" {
  description = "The address space that is used by the virtual network. You can supply more than one address space. Changing this forces a new resource to be created."
  default     = "10.0.0.0/16"
}

variable "vnet_spark_subnet1_name" {
  description = "The name used for the Master subnet."
  default     = "Subnet-Master"
}

variable "vnet_spark_subnet1_prefix" {
  description = "The address prefix to use for the Master subnet."
  default     = "10.0.0.0/24"
}

variable "vnet_spark_subnet2_name" {
  description = "The name used for the slave/agent subnet."
  default     = "Subnet-Slave"
}

variable "vnet_spark_subnet2_prefix" {
  description = "The address prefix to use for the slave/agent subnet."
  default     = "10.0.1.0/24"
}

variable "vnet_spark_subnet3_name" {
  description = "The name used for the subnet used by Cassandra."
  default     = "Subnet-Cassandra"
}

variable "vnet_spark_subnet3_prefix" {
  description = "The address prefix to use for the subnet used by Cassandra."
  default     = "10.0.2.0/24"
}

variable "nsg_spark_master_name" {
  description = "The name of the network security group for Spark's Master"
  default     = "nsg-spark-master"
}

variable "nsg_spark_slave_name" {
  description = "The name of the network security group for Spark's slave/agent nodes"
  default     = "nsg-spark-slave"
}

variable "nsg_cassandra_name" {
  description = "The name of the network security group for Cassandra"
  default     = "nsg-cassandra"
}

variable "nic_master_name" {
  description = "The name of the network interface card for Master"
  default     = "nic-master"
}

variable "nic_master_node_ip" {
  description = "The private IP address used by the Master's network interface card"
  default     = "10.0.0.5"
}

variable "nic_cassandra_name" {
  description = "The name of the network interface card used by Cassandra"
  default     = "nic-cassandra"
}

variable "nic_cassandra_node_ip" {
  description = "The private IP address of Cassandra's network interface card"
  default     = "10.0.2.5"
}

variable "nic_slave_name_prefix" {
  description = "The prefix used to constitute the slave/agents' names"
  default     = "nic-slave-"
}

variable "nic_slave_node_ip_prefix" {
  description = "The prefix of the private IP address used by the network interface card of the slave/agent nodes"
  default     = "10.0.1."
}

variable "public_ip_master_name" {
  description = "The name of the master node's public IP address"
  default     = "public-ip-master"
}

variable "public_ip_slave_name_prefix" {
  description = "The prefix to the slave/agent nodes' IP address names"
  default     = "public-ip-slave-"
}

variable "public_ip_cassandra_name" {
  description = "The name of Cassandra's node's public IP address"
  default     = "public-ip-cassandra"
}

variable "vm_master_name" {
  description = "The name of Spark's Master virtual machine"
  default     = "spark-master"
}

variable "vm_master_os_disk_name" {
  description = "The name of the os disk used by Spark's Master virtual machine"
  default     = "vmMasterOSDisk"
}

variable "vm_master_storage_account_container_name" {
  description = "The name of the storage account container used by Spark's master"
  default     = "vhds"
}

variable "vm_slave_name_prefix" {
  description = "The name prefix used by Spark's slave/agent nodes"
  default     = "spark-slave-"
}

variable "vm_slave_os_disk_name_prefix" {
  description = "The prefix used to constitute the names of the os disks used by the slave/agent nodes"
  default     = "vmSlaveOSDisk-"
}

variable "vm_slave_storage_account_container_name" {
  description = "The name of the storage account container used by the slave/agent nodes"
  default     = "vhds"
}

variable "vm_cassandra_name" {
  description = "The name of the virtual machine used by Cassandra"
  default     = "cassandra"
}

variable "vm_cassandra_os_disk_name" {
  description = "The name of the os disk used by the Cassandra virtual machine"
  default     = "vmCassandraOSDisk"
}

variable "vm_cassandra_storage_account_container_name" {
  description = "The name of the storage account container used by the Cassandra node"
  default     = "vhds"
}

variable "availability_slave_name" {
  description = "The name of the availability set for the slave/agent machines"
  default     = "availability-slave"
}

variable "script_spark_provisioner_script_file_name" {
  description = "The name of the script kept in version control which will provision Spark"
  default     = "scriptSparkProvisioner.sh"
}

variable "script_cassandra_provisioner_script_file_name" {
  description = "The name of the script kept in version control which will provision Cassandra"
  default     = "scriptCassandraProvisioner.sh"
}
