variable "availability_zones" {
  default = "cn-beijing-c"
}

variable "name" {
  default = "slb_alicloud"
}

variable "cidr_blocks" {
  type = "map"
  default = {
    az0 = "10.1.1.0/24"
    az1 = "10.1.2.0/24"
    az2 = "10.1.3.0/24"
  }
}

variable "internet_charge_type" {
  default = "paybytraffic"
}

variable "long_name" {
  default = "alicloud"
}
variable "vpc_cidr" {
  default = "10.1.0.0/21"
}
variable "region" {
  default = "cn-beijing"
}