variable "dyn_src" {
  type  = string
  const = true
}

terraform {
  required_providers {
    dynamic = {
      source = var.dyn_src
    }
    static = {
      source = "hashicorp/static"
    }
  }
}
