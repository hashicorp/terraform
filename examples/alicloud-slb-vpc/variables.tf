variable "name" {
  default = "slb_alicloud"
}
variable "vpc_id" {
  default = "vpc-2ze0z1hayvlsbk98gw805"
}
variable "vswitch_id" {
  default = "vsw-2ze7cfya11g7uah2grc8f"
}

variable "instances" {
  type = "list"
  default = [
    "i-2zecejialx1rx513qcyv",
    "i-2zedgb871dbnpc5x3w9n"]
}

variable "internet_charge_type" {
  default = "paybytraffic"
}

variable "internet" {
  default = "false"
}
