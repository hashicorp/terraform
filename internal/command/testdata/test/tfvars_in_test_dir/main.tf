variable "foo" {
  description = "This is test variable"
  default = "def_value"
}

variable "fooJSON" {
  description = "This is test variable"
  default = "def_value"
}

output "out_foo" {
  value = var.foo
}

output "out_fooJSON" {
  value = var.fooJSON
}
