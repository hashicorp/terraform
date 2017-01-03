variable "count" {
  default = "6"
}
variable "count_format" {
  default = "%02d"
}

variable "security_groups" {
  type = "list"
  default = ["sg-2zecd09tw30jo1c7ekdi"]
}
variable "ecs_password" {
  default = "Test12345"
}
variable "slb_id"{
  default = "lb-2zel5fjqk1qgmwud7t3xb"
}