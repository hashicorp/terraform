variable "count" {
  default = "1"
}
variable "count_format" {
  default = "%02d"
}
variable "image_id" {
  default = "ubuntu1404_64_40G_cloudinit_20160727.raw"
}

variable "role" {
  default = "worder"
}
variable "datacenter" {
  default = "beijing"
}
variable "short_name" {
  default = "hi"
}
variable "ecs_type" {
  default = "ecs.n1.small"
}
variable "ecs_password" {
  default = "Test12345"
}
variable "availability_zones" {
  default = "cn-beijing-b"
}
variable "security_group_id" {
  default = "sg-25y6ag32b"
}
variable "ssh_username" {
  default = "root"
}

variable "internet_charge_type" {
  default = "PayByTraffic"
}

variable "slb_internet_charge_type" {
  default = "paybytraffic"
}
variable "instance_network_type" {
  default = "Classic"
}
variable "internet_max_bandwidth_out" {
  default = 5
}

variable "disk_category" {
  default = "cloud_ssd"
}
variable "disk_size" {
  default = "40"
}
variable "device_name" {
  default = "/dev/xvdb"
}

variable "slb_name" {
  default = "slb_worder"
}

variable "internet" {
  default = true
}

variable "load_balancer_weight" {
  default = "100"
}