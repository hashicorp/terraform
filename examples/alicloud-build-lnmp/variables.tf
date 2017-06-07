variable "region" {
  default = "cn-beijing"
}
variable "vpc_cidr" {
  default = "10.1.0.0/21"
}
variable "vswitch_cidr" {
  default = "10.1.1.0/24"
}
variable "io_optimized" {
  default = "optimized"
}
variable "ecs_password" {
  default = "Test1234567*"
}
variable "disk_category" {
  default = "cloud_efficiency"
}
variable "db_name" {
  default = "lnmp"
}
variable "db_user" {
  default = "alier"
}
variable "db_password" {
  default = "123456"
}
variable "db_root_password" {
  default = "123456"
}