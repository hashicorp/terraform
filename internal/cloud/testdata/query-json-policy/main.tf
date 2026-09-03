# This directory tests policy-bearing query requests and summary rendering.
# The query configuration is driven by main.tfquery.hcl; there are no
# Terraform-managed resources in this fixture.

variable "foo" {
  type    = string
  default = ""
}

variable "bar" {
  type    = string
  default = ""
}
