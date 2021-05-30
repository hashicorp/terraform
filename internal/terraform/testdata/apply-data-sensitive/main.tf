variable "foo" {
  sensitive = true
  default = "foo"
}

data "null_data_source" "testing" {
  foo = var.foo
}
