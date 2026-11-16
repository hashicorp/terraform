
variable "value" {
  type = string
default = "value"
}

list foo_list "some_list_block" {
  provider=foo

  config {
  condition = var.value
      filter = 42
  }
}
