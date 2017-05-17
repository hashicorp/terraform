variable "count" {
  default = "1"
}
variable "count_format" {
  default = "%02d"
}
variable "most_recent" {
  default = true
}

variable "image_owners" {
  default = ""
}

variable "name_regex" {
  default = "^centos_6\\w{1,5}[64].*"
}

variable "role" {
  default = "work"
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
variable "allocate_public_ip" {
  default = true
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
variable "disk_category" {
  default = "cloud_ssd"
}
variable "disk_size" {
  default = "40"
}
variable "device_name" {
  default = "/dev/xvdb"
}