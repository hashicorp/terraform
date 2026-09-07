
required_providers {
  null = {
    source  = "hashicorp/null"
    version = "3.2.1"
  }
}

variable "name" {
  type = string
}

component "a" {
  source = "./a"

  inputs = {
    name = var.name
  }
  providers = {
    null = var.provider
  }
}
