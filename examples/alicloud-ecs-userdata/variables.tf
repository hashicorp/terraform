variable "vpc_cidr" {
  default = "172.16.0.0/12"
}

variable "vswitch_cidr" {
  default = "172.16.0.0/21"
}

variable "zone" {
  default = "cn-beijing-a"
}

variable "password" {
  default = "Test123456"
}

variable "image" {
  default = "ubuntu_140405_32_40G_cloudinit_20161115.vhd"
}

variable "ecs_type" {
  default = "ecs.n4.large"
}