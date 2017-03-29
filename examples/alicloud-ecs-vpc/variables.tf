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
  default = "Test12345"
}
variable "availability_zones" {
}
variable "security_groups" {
  type    = "list"
}
variable "ssh_username" {
  default = "root"
}

//if instance_charge_type is "PrePaid", then must be set period, the value is 1 to 30, unit is month
variable "instance_charge_type" {
  default = "PostPaid"
}

variable "system_disk_category" {
  default = "cloud_efficiency"
}

variable "internet_charge_type" {
  default = "PayByTraffic"
}
variable "internet_max_bandwidth_out" {
  default = 5
}

variable "io_optimized" {
  default = "optimized"
}

variable "allocate_public_ip" {
  default = true
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

variable "vswitch_id" {
  default = ""
}