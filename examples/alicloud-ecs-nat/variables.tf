variable "vpc_cidr" {
  default = "10.1.0.0/21"
}

variable "vswitch_cidr" {
  default = "10.1.1.0/24"
}

variable "zone" {
  default = "cn-beijing-c"
}

variable "image" {
  default = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
}

variable "instance_nat_type" {
  default = "ecs.n1.small"
}

variable "instance_worker_type" {
  default = "ecs.s2.large"
}

variable "instance_pwd" {
  default = "Test123456"
}