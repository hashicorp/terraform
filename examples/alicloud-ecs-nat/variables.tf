variable "vpc_cidr" {
  default = "10.1.0.0/21"
}

variable "vswitch_cidr" {
  default = "10.1.1.0/24"
}

variable "zone" {
  default = "cn-beijing-d"
}

variable "image" {
  default = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
}

variable "instance_nat_type" {
  default = "ecs.n4.small"
}

variable "instance_worker_type" {
  default = "ecs.n4.large"
}

variable "instance_pwd" {
  default = "Test123456"
}