variable "ecs_password" {
  default = "Test12345"
}

variable "control_count" {
  default = "3"
}
variable "control_count_format" {
  default = "%02d"
}
variable "control_ecs_type" {
  default = "ecs.n1.medium"
}
variable "control_disk_size" {
  default = "100"
}

variable "edge_count" {
  default = "2"
}
variable "edge_count_format" {
  default = "%02d"
}
variable "edge_ecs_type" {
  default = "ecs.n1.small"
}

variable "worker_count" {
  default = "1"
}
variable "worker_count_format" {
  default = "%03d"
}
variable "worker_ecs_type" {
  default = "ecs.n1.small"
}

variable "short_name" {
  default = "ali"
}
variable "ssh_username" {
  default = "root"
}

variable "region" {
  default = "cn-beijing"
}

variable "availability_zones" {
  default = "cn-beijing-c"
}

variable "datacenter" {
  default = "beijing"
}