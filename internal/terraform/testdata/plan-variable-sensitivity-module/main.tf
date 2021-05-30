variable "sensitive_var" {
  default   = "foo"
  sensitive = true
}

variable "another_var" {
  sensitive = true
}

module "child" {
  source = "./child"
  foo    = var.sensitive_var
  bar    = var.another_var
}
