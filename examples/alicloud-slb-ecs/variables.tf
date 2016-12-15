variable "slb_name" {
  default = "slb_worker"
}

variable "instances" {
  type = "list"
  default = [
    "i-2ze8q011etu3a54eym2u",
    "i-2zebe1ftlybmza8dfmyf"]

}

variable "internet_charge_type" {
  default = "paybytraffic"
}

variable "internet" {
  default = true
}