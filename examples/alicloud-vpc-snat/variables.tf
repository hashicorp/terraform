
variable "vpc_cidr" {
  default = "10.1.0.0/21"
}
variable "vswitch_cidr" {
  default = "10.1.1.0/24"
}
variable "rule_policy" {
  default = "accept"
}
variable "instance_type" {
  default = "ecs.n4.small"
}
variable "image_id" {
  default = "ubuntu_140405_64_40G_cloudinit_20161115.vhd"
}
variable "disk_category"{
  default = "cloud_efficiency"
}