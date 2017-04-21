variable "count" {
  default = "1"
}
variable "count_format" {
  default = "%02d"
}
variable "image_id" {
  default = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
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
variable "ssh_username" {
  default = "root"
}

variable "allocate_public_ip" {
  default = true
}

variable "internet_charge_type" {
  default = "PayByTraffic"
}

variable "slb_internet_charge_type" {
  default = "paybytraffic"
}
variable "internet_max_bandwidth_out" {
  default = 5
}

variable "io_optimized" {
  default = "optimized"
}

variable "slb_name" {
  default = "slb_worder"
}

variable "internet" {
  default = true
}
