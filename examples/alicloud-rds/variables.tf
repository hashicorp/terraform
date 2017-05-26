variable "engine" {
  default = "MySQL"
}
variable "engine_version" {
  default = "5.6"
}
variable "instance_class" {
  default = "rds.mysql.t1.small"
}
variable "storage" {
  default = "10"
}
variable "net_type" {
  default = "Intranet"
}

variable "user_name" {
  default = "tf_tester"
}
variable "password" {
  default = "Test12345"
}

variable "database_name" {
  default = "bookstore"
}
variable "database_character" {
  default = "utf8"
}