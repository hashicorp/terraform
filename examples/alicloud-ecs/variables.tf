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
}
variable "datacenter" {
}
variable "short_name" {
  default = "hi"
}
variable "ecs_type" {
}
variable "ecs_password" {
}
variable "availability_zones" {
}
variable "allocate_public_ip" {
  default = true
}
variable "internet_charge_type" {
  default = "PayByTraffic"
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