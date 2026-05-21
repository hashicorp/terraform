terraform {
  cloud {
    organization = "hashicorp"
    workspaces {
      name = "test"
    }
  }
}

variable "module_name" {
  type  = string
  const = true
}

module "example" {
  source = "./modules/${var.module_name}"
}
