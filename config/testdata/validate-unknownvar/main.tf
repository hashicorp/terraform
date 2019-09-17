variable "foo" {
    default = "bar"
    description = "bar"
}

provider "do" {
  api_key = "${var.bar}"
}
