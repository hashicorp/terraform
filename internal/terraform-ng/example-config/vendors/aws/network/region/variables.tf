
variable "region_name" {
  type = string
}

variable "base_cidr_block" {
  type = string

  validation {
    condition     = can(cidrhost(var.base_cidr_block, 0)) && can(regex("^[\\d\\./]+$"))
    error_message = "Must be an IPv4 CIDR prefix given as ADDRESS/LENGTH."
  }
}

variable "availability_zones" {
  type = list(string)
}

variable "max_availability_zones" {
  type = number
}

variable "common_tags" {
  name = map(string)
}
