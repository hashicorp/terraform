terraform {
  experiments = [sensitive_variables]
}

variable "sensitive_var" {
  default   = "foo"
  sensitive = true
}

module "child" {
  source = "./child"
  foo    = var.sensitive_var
}
